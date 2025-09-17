package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bakito/policy-report-publisher/internal/ingest"
	"github.com/bakito/policy-report-publisher/internal/report"
)

func main() {
	// Build base context that cancels on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Create report handler (updates PolicyReports)
	h, err := report.NewHandler()
	if err != nil {
		slog.Error("failed to create report handler", "error", err)
		os.Exit(1)
	}

	// Shared items channel for all ingest paths
	itemCh := make(chan *report.Item, 1024)

	// Start gRPC ingest endpoint (sidecars send items here)
	grpcAddr := os.Getenv("PUBLISHER_GRPC_ADDR")
	if grpcAddr == "" {
		grpcAddr = "127.0.0.1:9090"
	}
	go func() {
		if err := ingest.StartGRPC(ctx, grpcAddr, itemCh); err != nil {
			slog.Error("grpc ingest server failed", "addr", grpcAddr, "error", err)
			// If server fails unexpectedly, stop main context
			stop()
		}
	}()

	// Optionally start HTTP ingest (if enabled via env)
	if httpAddr := os.Getenv("PUBLISHER_HTTP_ADDR"); httpAddr != "" {
		go func() {
			if err := ingest.StartHTTP(ctx, httpAddr, itemCh); err != nil {
				slog.Error("http ingest server failed", "addr", httpAddr, "error", err)
				stop()
			}
		}()
	}

	// Worker to process items and update PolicyReports
	workerCount := 4
	for i := 0; i < workerCount; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case it := <-itemCh:
					if it == nil {
						continue
					}
					// Best-effort retry on transient conflicts is already handled inside Update
					if err := h.Update(ctx, it); err != nil {
						slog.Warn("failed to update PolicyReport", "namespace", it.Namespace, "name", it.Name, "error", err)
						// small delay to avoid hot-looping on persistent failures
						time.Sleep(200 * time.Millisecond)
					}
				}
			}
		}()
	}

	// Block until context is cancelled
	<-ctx.Done()
	slog.Info("shutting down")
	// allow in-flight updates to finish briefly
	time.Sleep(500 * time.Millisecond)
}
