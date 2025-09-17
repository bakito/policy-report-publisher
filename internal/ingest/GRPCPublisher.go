package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/bakito/policy-report-publisher/pkg/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewGRPCPublisher connects to the publisher's gRPC ingest server using the JSON codec.
// It returns a function that accepts a batch of items and a Close function.
//
// Usage in a sidecar-ish adapter process:
//
//	pub, close, _ := ingest.NewGRPCPublisher(ctx, "127.0.0.1:9090")
//	defer close()
//	_ = pub([]*api.Item{ item1, item2 })
func NewGRPCPublisher(ctx context.Context, addr string) (publish func(items []*api.Item) error, close func() error, err error) {
	RegisterJSONCodec()

	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()), // plaintext in-pod; replace with TLS for cross-pod traffic
		grpc.WithDefaultCallOptions(grpc.ForceCodec(jsonCodec{})),
		grpc.WithContextDialer(defaultDialer),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("grpc dial: %w", err)
	}

	invoke := func(items []*api.Item) error {
		wire := make([]*wireItem, 0, len(items))
		for _, it := range items {
			if it == nil {
				continue
			}
			var raw json.RawMessage
			if src := it.Source; src != nil {
				if b, err := json.Marshal(src); err == nil {
					raw = json.RawMessage(b)
				}
			}
			wire = append(wire, &wireItem{
				Namespace: it.Namespace,
				Name:      it.Name,
				HandlerID: it.HandlerID,
				Result:    it.Result,
				Source:    raw,
			})
		}
		req := &IngestItems{Items: wire}
		var resp Ack
		return conn.Invoke(ctx, "/policyreport.publisher.v1.IngestService/PushItems", req, &resp)
	}
	return invoke, conn.Close, nil
}

// PushChannel creates a channel that automatically batches items and pushes them
// to the server in the background. Close the returned function to flush and stop.
//
// Example:
//
//	ch, stop, _ := ingest.PushChannel(ctx, "127.0.0.1:9090", 100, time.Second)
//	defer stop()
//	ch <- item
func PushChannel(ctx context.Context, addr string, maxBatch int, maxWait time.Duration) (chan<- *api.Item, func() error, error) {
	pub, closeConn, err := NewGRPCPublisher(ctx, addr)
	if err != nil {
		return nil, nil, err
	}

	in := make(chan *api.Item, maxBatch*2)
	stopCh := make(chan struct{})

	go func() {
		defer func() {
			_ = closeConn()
		}()

		batch := make([]*api.Item, 0, maxBatch)
		flush := func() {
			if len(batch) == 0 {
				return
			}
			_ = pub(batch)
			batch = batch[:0]
		}

		t := time.NewTimer(maxWait)
		defer t.Stop()

		for {
			select {
			case it, ok := <-in:
				if !ok {
					flush()
					return
				}
				if it != nil {
					batch = append(batch, it)
					if len(batch) >= maxBatch {
						flush()
						if !t.Stop() {
							select {
							case <-t.C:
							default:
							}
						}
						t.Reset(maxWait)
					}
				}
			case <-t.C:
				flush()
				t.Reset(maxWait)
			case <-ctx.Done():
				flush()
				return
			case <-stopCh:
				flush()
				return
			}
		}
	}()

	return in, func() error { close(stopCh); return nil }, nil
}

// defaultDialer keeps grpc.Dial from doing DNS lookups for plain host:port on loopback.
// It can be customized if you need SOCKS, HTTP CONNECT, etc.
func defaultDialer(ctx context.Context, addr string) (netConn net.Conn, err error) {
	d := &net.Dialer{}
	return d.DialContext(ctx, "tcp", addr)
}
