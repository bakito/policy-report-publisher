package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/bakito/policy-report-publisher/pkg/hubble"
	"github.com/bakito/policy-report-publisher/pkg/kubearmor"
	"github.com/bakito/policy-report-publisher/pkg/report"
	"github.com/bakito/policy-report-publisher/version"
)

func init() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
}

func main() {
	var withHubble bool
	var withKubeArmor bool

	flag.BoolVar(&withHubble, "hubble", false, "enable hubble")
	flag.BoolVar(&withKubeArmor, "kubearmor", false, "enable kubearmor")
	flag.Parse()

	if !withKubeArmor && !withHubble {
		slog.Error("either 'hubble' or 'kubearmor' must be enabled")
		os.Exit(1)
	}

	slog.Info("policy-report-publisher", "version", version.Version, "hubble", withHubble, "kubearmor", withKubeArmor)

	// Initialize the report handler
	handler, err := report.NewHandler()
	if err != nil {
		slog.Error("failed to create report handler", "error", err)
		os.Exit(1)
	}

	// Create a cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a channel for reports
	reportChan := make(chan *report.Item, 100) // Buffered for performance

	// Handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		slog.Info("Shutting down gracefully...")
		cancel()
	}()

	if withKubeArmor {
		go func() {
			if err := kubearmor.Run(ctx, reportChan); err != nil {
				slog.Error("kubearmor.Run exited with error", "error", err)
				cancel()
			}
		}()
	}
	if withHubble {
		go func() {
			if err := hubble.Run(ctx, reportChan); err != nil {
				slog.Error("hubble.Run exited with error", "error", err)
				cancel()
			}
		}()
	}

	// Process reports from the channel
	for {
		select {
		case report, ok := <-reportChan:
			if !ok {
				// Channel closed, exit loop
				return
			}
			if err := handler.Update(ctx, report); err != nil {
				slog.Error("Failed to update report", "error", err)
			}
		case <-ctx.Done():
			// Context is done, exit loop
			slog.Info("Context done, exiting report processing loop.")
			return
		}
	}
}
