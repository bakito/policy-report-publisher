package ingest

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/bakito/policy-report-publisher/pkg/api"
)

// StartHTTP starts an HTTP server that ingests report items from sidecars.
// - addr: e.g. ":8080" or "127.0.0.1:8080". It is recommended to bind to loopback.
// - ch: channel where decoded report items are sent.
// The server shuts down when ctx is done.
func StartHTTP(ctx context.Context, addr string, ch chan *api.Item) error {
	mux := http.NewServeMux()
	mux.Handle("/healthz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	mux.Handle("/readyz", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	}))

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Ensure we bind to a loopback or specified addr
	lis, err := (&net.ListenConfig{}).Listen(ctx, "tcp", addr)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("ingest http shutdown error", "error", err)
		}
	}()

	slog.Info("sidecar ingest http server starting", "addr", addr)
	if err := srv.Serve(lis); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	slog.Info("sidecar ingest http server stopped")
	return nil
}
