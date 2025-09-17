package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
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
	mux.Handle("/v1/items", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ct := r.Header.Get("Content-Type")
		if ct != "" && !strings.Contains(ct, "application/json") {
			http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
			return
		}
		defer r.Body.Close()

		body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20)) // 10MiB limit
		if err != nil {
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}
		trimmed := strings.TrimSpace(string(body))
		if len(trimmed) == 0 {
			http.Error(w, "empty body", http.StatusBadRequest)
			return
		}

		// Accept either a single object or an array of objects.
		if strings.HasPrefix(trimmed, "[") {
			var items []*api.Item
			if err := json.Unmarshal([]byte(trimmed), &items); err != nil {
				http.Error(w, "invalid JSON array: "+err.Error(), http.StatusBadRequest)
				return
			}
			count := 0
			for _, it := range items {
				if it == nil {
					continue
				}
				select {
				case ch <- it:
					count++
				case <-ctx.Done():
					http.Error(w, "server shutting down", http.StatusServiceUnavailable)
					return
				}
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"accepted":` + itoa(count) + `}`))
			return
		}

		var item api.Item
		if err := json.Unmarshal([]byte(trimmed), &item); err != nil {
			http.Error(w, "invalid JSON object: "+err.Error(), http.StatusBadRequest)
			return
		}
		select {
		case ch <- &item:
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"accepted":1}`))
		case <-ctx.Done():
			http.Error(w, "server shutting down", http.StatusServiceUnavailable)
		}
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

// itoa: small helper to avoid pulling strconv for a single call.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
