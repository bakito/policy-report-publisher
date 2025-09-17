package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/bakito/hubble-policy-report/internal/hubble"
	"github.com/bakito/policy-report-publisher/pkg/adapter"
)

func main() {
	if err := adapter.Start(context.Background(), hubble.Run, os.Getenv("PUBLISHER_GRPC_ADDR")); err != nil {
		slog.Error("failed to start report handler", "error", err)
	}
}
