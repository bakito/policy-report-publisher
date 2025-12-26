package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"plugin"
	"strings"
	"syscall"

	"github.com/bakito/policy-report-publisher-shared/env"
	"github.com/bakito/policy-report-publisher-shared/types"
	"github.com/bakito/policy-report-publisher/internal/metrics"
	"github.com/bakito/policy-report-publisher/internal/report"
	"github.com/bakito/policy-report-publisher/version"
	"k8s.io/klog/v2"
)

const (
	LogReports       = "LOG_REPORTS"
	LeaderElectionNS = "LEADER_ELECTION_NAMESPACE"
)

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	klog.SetSlogLogger(logger)

	slog.InfoContext(ctx, "policy-report-publisher", "version", version.Version,
		"log-reports", env.Active(LogReports))

	// Initialize the report handler
	handler, err := report.NewHandler()
	if err != nil {
		slog.ErrorContext(ctx, "failed to create report handler", "error", err)
		os.Exit(1)
	}

	ok, err := handler.PolicyReportAvailable()
	if err != nil {
		slog.ErrorContext(ctx, "could not check if PolicyReport is available", "error", err)
		os.Exit(1)
	}
	// https://github.com/kubernetes-sigs/wg-policy-prototypes/blob/25056e1f3eb5cab1e693b8c880eb693a84e099af/policy-report/crd/v1beta2/wgpolicyk8s.io_policyreports.yaml
	if !ok {
		slog.ErrorContext(ctx, "PolicyReport CRD is not available, please install kyverno",
			"APIVersion", report.PolicyReport.APIVersion,
			"Kind", report.PolicyReport.Kind,
		)
		os.Exit(1)
	}

	// Create a cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go metrics.Start(ctx)

	if ns, ok := os.LookupEnv(LeaderElectionNS); ok && strings.TrimSpace(ns) != "" {
		if err := handler.RunAsLeader(ctx, cancel, ns, run); err != nil {
			slog.ErrorContext(ctx, "error running with leader election", "error", err)
			os.Exit(1)
		}
	} else {
		run(ctx, handler, cancel)
	}
}

func run(ctx context.Context, handler report.Handler, cancel context.CancelFunc) {
	// Create a channel for reports
	reportChan := make(chan *types.Item, 100) // Buffered for performance

	// Handle OS signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		slog.InfoContext(ctx, "Shutting down gracefully...")
		cancel()
	}()

	startPlugins(ctx, reportChan, cancel)

	// Process reports from the channel
	for {
		select {
		case rep, ok := <-reportChan:
			if !ok {
				// Channel closed, exit loop
				return
			}
			if err := handler.Update(ctx, rep); err != nil {
				slog.ErrorContext(ctx, "Failed to update report", "error", err)
			}
		case <-ctx.Done():
			// Context is done, exit loop
			slog.InfoContext(ctx, "Context done, exiting report processing loop.")
			return
		}
	}
}

func startPlugins(ctx context.Context, reportChan chan *types.Item, cancel context.CancelFunc) {
	plugins, err := os.ReadDir("plugins")
	if err != nil {
		if os.IsNotExist(err) {
			slog.InfoContext(ctx, "no plugins directory found, skipping plugin loading")
			return
		}
		slog.ErrorContext(ctx, "error reading plugins directory", "error", err)
		return
	}

	for _, p := range plugins {
		if !p.IsDir() && strings.HasSuffix(p.Name(), ".so") {
			pluginPath := "plugins/" + p.Name()
			plug, err := plugin.Open(pluginPath)
			if err != nil {
				slog.ErrorContext(ctx, "error opening plugin", "path", pluginPath, "error", err)
				continue
			}

			runSymbol, err := plug.Lookup("Run")
			if err != nil {
				slog.ErrorContext(ctx, "error looking up Run symbol in plugin", "path", pluginPath, "error", err)
				continue
			}

			runFunc, ok := runSymbol.(func(context.Context, chan *types.Item) error)
			if !ok {
				slog.ErrorContext(ctx, "invalid Run function signature in plugin", "path", pluginPath)
				continue
			}

			pluginName := strings.TrimSuffix(p.Name(), ".so")
			start(ctx, reportChan, cancel, pluginName, "PLUGIN_"+pluginName, runFunc)
		}
	}
}

func start(ctx context.Context,
	reportChan chan *types.Item,
	cancel context.CancelFunc,
	name string,
	serviceVar string,
	run func(ctx context.Context, reportChan chan *types.Item) error,
) {
	if strings.HasPrefix(serviceVar, "PLUGIN_") || !env.Empty(serviceVar) {
		go func() {
			slog.InfoContext(ctx, "starting", "name", name, "service", os.Getenv(serviceVar))
			if err := run(ctx, reportChan); err != nil {
				slog.ErrorContext(ctx, "run exited with error", "name", name, "error", err)
				cancel()
			}
		}()
	}
}
