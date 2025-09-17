package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/bakito/kubearmor-policy-report/internal/kubearmor"
	"github.com/bakito/policy-report-publisher/pkg/adapter"
)

func main() {
	if err := adapter.Start(context.Background(), kubearmor.New(), os.Getenv("PUBLISHER_GRPC_ADDR")); err != nil {
		slog.Error("failed to start report handler", "error", err)
	}
}
