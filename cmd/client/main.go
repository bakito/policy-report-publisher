// go
package main

import (
	"context"
	"log"
	"os"

	"github.com/bakito/policy-report-publisher/internal/adapter/dummy"
	"github.com/bakito/policy-report-publisher/internal/adapter/hubble"
	"github.com/bakito/policy-report-publisher/internal/adapter/kubearmor"
	"github.com/bakito/policy-report-publisher/internal/ingest"
	"github.com/bakito/policy-report-publisher/internal/report"
)

func main() {
	ctx := context.Background()

	addr := os.Getenv("PUBLISHER_GRPC_ADDR")
	if addr == "" {
		addr = "127.0.0.1:9090"
	}

	// Connect a gRPC publisher to the server
	pub, closeConn, err := ingest.NewGRPCPublisher(ctx, addr)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = closeConn() }()

	// Bridge: create a bidirectional channel for the adapter and send directly to the server via gRPC.
	adapterCh := make(chan *report.Item, 200)
	defer close(adapterCh)
	go func() {
		for it := range adapterCh {
			if it == nil {
				continue
			}
			// Send each item to the server; you can batch if desired.
			_ = pub([]*report.Item{it})
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()

	// Choose which adapter to run for this sidecar via env, args, or container image.
	var runAdapter func(context.Context, chan *report.Item) error
	switch os.Getenv("ADAPTER") {
	case "hubble":
		runAdapter = hubble.Run
	case "kubearmor":
		runAdapter = kubearmor.Run
	case "dummy":
		runAdapter = dummy.Run
	default:
		log.Fatal("unknown or empty ADAPTER; set ADAPTER=hubble|kube-armor|dummy")
	}

	if err := runAdapter(ctx, adapterCh); err != nil {
		log.Fatal(err)
	}
}
