package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"time"

	"github.com/bakito/policy-report-publisher/pkg/api"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// IngestItems represents a batch pushed by the sidecar.
type IngestItems struct {
	Items []*wireItem `json:"items"`
}

type wireItem struct {
	Namespace string           `json:"namespace"`
	Name      string           `json:"name"`
	HandlerID string           `json:"handlerId"`
	Result    api.ReportResult `json:"result"`
	Source    json.RawMessage  `json:"source,omitempty"`
}

// Ack is a simple acknowledgement payload.
type Ack struct {
	Accepted int `json:"accepted"`
}

// IngestService provides methods for sidecars to push items.
type IngestService interface {
	// PushItems ingests a batch; unary, simple to use from sidecars.
	PushItems(ctx context.Context, req *IngestItems) (*Ack, error)
	// StreamItems ingests a stream of batches; optional for long-lived sidecars.
	StreamItems(stream grpc.ServerStream) error
}

type ingestServer struct {
	ch chan *api.Item
}

func newIngestServer(ch chan *api.Item) *ingestServer {
	return &ingestServer{ch: ch}
}

func (s *ingestServer) PushItems(ctx context.Context, req *IngestItems) (*Ack, error) {
	if req == nil {
		return &Ack{Accepted: 0}, nil
	}
	accepted := 0
	for _, wi := range req.Items {
		if wi == nil {
			continue
		}
		var src any
		if len(wi.Source) > 0 && string(wi.Source) != "null" {
			// keep as generic map to avoid tight coupling
			var m map[string]any
			if err := json.Unmarshal(wi.Source, &m); err == nil {
				src = m
			} else {
				// fallback: store as raw json string
				src = wi.Source
			}
		}
		it := api.ItemFor(wi.HandlerID, wi.Namespace, wi.Name, wi.Result, src)

		select {
		case s.ch <- it:
			accepted++
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return &Ack{Accepted: accepted}, nil
}

// Stream request/response envelopes for the manual grpc.ServiceDesc registration.
type (
	streamItemsReq = IngestItems
	streamItemsAck = Ack
)

func (s *ingestServer) StreamItems(stream grpc.ServerStream) error {
	accepted := 0
	for {
		var req streamItemsReq
		if err := stream.RecvMsg(&req); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			// EOF or transport close ends the stream gracefully.
			return nil
		}

		for _, wi := range req.Items {
			if wi == nil {
				continue
			}
			var src any
			if len(wi.Source) > 0 && string(wi.Source) != "null" {
				var m map[string]any
				if err := json.Unmarshal(wi.Source, &m); err == nil {
					src = m
				} else {
					src = wi.Source
				}
			}
			it := api.ItemFor(wi.HandlerID, wi.Namespace, wi.Name, wi.Result, src)

			// stream.Context() is bound to this client-stream
			select {
			case s.ch <- it:
				accepted++
			case <-stream.Context().Done():
				return stream.Context().Err()
			}
		}
		// Optionally send intermediate acknowledgements.
		if err := stream.SendMsg(&streamItemsAck{Accepted: accepted}); err != nil {
			return err
		}
	}
}

// Service description for manual registration (no .proto dependency).
var _IngestServiceDesc = grpc.ServiceDesc{
	ServiceName: "policyreport.publisher.v1.IngestService",
	HandlerType: (*IngestService)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "PushItems",
			Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
				in := new(IngestItems)
				if err := dec(in); err != nil {
					return nil, err
				}
				if interceptor == nil {
					return srv.(*ingestServer).PushItems(ctx, in)
				}
				info := &grpc.UnaryServerInfo{
					Server:     srv,
					FullMethod: "/policyreport.publisher.v1.IngestService/PushItems",
				}
				handler := func(ctx context.Context, req any) (any, error) {
					return srv.(*ingestServer).PushItems(ctx, req.(*IngestItems))
				}
				return interceptor(ctx, in, info, handler)
			},
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamItems",
			Handler:       _streamItemsHandler,
			ServerStreams: true, // server sends acks
			ClientStreams: true, // client sends reqs
		},
	},
}

func _streamItemsHandler(srv any, stream grpc.ServerStream) error {
	return srv.(*ingestServer).StreamItems(stream)
}

// StartGRPC unchanged below...
// ... existing code ...
// StartGRPC starts the JSON-over-gRPC ingest server.
// - addr: e.g., ":9090" or "127.0.0.1:9090"
// - ch: the same channel you use in your pipeline to publish *api.Item
func StartGRPC(ctx context.Context, addr string, ch chan *api.Item) error {
	RegisterJSONCodec()

	lis, err := (&net.ListenConfig{}).Listen(ctx, "tcp", addr)
	if err != nil {
		return err
	}

	hs := health.NewServer()
	s := grpc.NewServer(
		// Enforce the JSON codec so clients can be very small and not rely on generated code.
		grpc.ForceServerCodec(jsonCodec{}),
	)
	healthpb.RegisterHealthServer(s, hs)

	srv := newIngestServer(ch)
	s.RegisterService(&_IngestServiceDesc, srv)

	// Graceful shutdown on ctx.Done()
	go func() {
		<-ctx.Done()
		done := make(chan struct{})
		go func() {
			s.GracefulStop()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			s.Stop()
		}
	}()

	slog.Info("sidecar ingest gRPC server starting", "addr", addr)
	if err := s.Serve(lis); err != nil {
		// If Serve returns because Stop/GracefulStop was called, swallow the error.
		select {
		case <-ctx.Done():
			return nil
		default:
			return err
		}
	}
	slog.Info("sidecar ingest gRPC server stopped")
	return nil
}
