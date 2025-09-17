package adapter

import (
	"context"

	"github.com/bakito/policy-report-publisher/internal/ingest"
	"github.com/bakito/policy-report-publisher/pkg/api"
)

type Adapter func(context.Context, chan *api.Item) error

func Start(ctx context.Context, adapter Adapter, publisherAddr string) error {
	if publisherAddr == "" {
		publisherAddr = "127.0.0.1:9090"
	}

	// Connect a gRPC publisher to the server
	pub, closeConn, err := ingest.NewGRPCPublisher(ctx, publisherAddr)
	if err != nil {
		return err
	}
	defer func() { _ = closeConn() }()

	// Bridge: create a bidirectional channel for the adapter and send directly to the server via gRPC.
	adapterCh := make(chan *api.Item, 200)
	defer close(adapterCh)
	go func() {
		for it := range adapterCh {
			if it == nil {
				continue
			}
			// Send each item to the server; you can batch if desired.
			_ = pub([]*api.Item{it})
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()

	return adapter(ctx, adapterCh)
}
