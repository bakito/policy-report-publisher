package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bakito/policy-report-publisher/pkg/adapter/hubble"
	"github.com/bakito/policy-report-publisher/pkg/adapter/kubearmor"
	"github.com/bakito/policy-report-publisher/pkg/env"
	"github.com/bakito/policy-report-publisher/pkg/metrics"
	"github.com/bakito/policy-report-publisher/pkg/report"
	"github.com/bakito/policy-report-publisher/version"
	"k8s.io/klog/v2"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	klog.SetSlogLogger(logger)

	if env.Empty(env.HubbleServiceName) && env.Empty(env.KubeArmorServiceName) {
		slog.Error("either 'Hubble' or 'KubeArmor' must be enabled",
			"hubble", env.HubbleServiceName,
			"kubearmor", env.KubeArmorServiceName)
		os.Exit(1)
	}

	slog.Info("policy-report-publisher", "version", version.Version,
		"hubble", os.Getenv(env.HubbleServiceName),
		"kubearmor", os.Getenv(env.KubeArmorServiceName),
		"log-reports", env.Active(env.LogReports))

	// Initialize the report handler
	handler, err := report.NewHandler()
	if err != nil {
		slog.Error("failed to create report handler", "error", err)
		os.Exit(1)
	}

	ok, err := handler.PolicyReportAvailable()
	if err != nil {
		slog.Error("could not check if PolicyReport is available", "error", err)
		os.Exit(1)
	}
	// https://github.com/kubernetes-sigs/wg-policy-prototypes/blob/25056e1f3eb5cab1e693b8c880eb693a84e099af/policy-report/crd/v1beta2/wgpolicyk8s.io_policyreports.yaml
	if !ok {
		slog.Error("PolicyReport CRD is not available, please install kyverno",
			"APIVersion", report.PolicyReport.APIVersion,
			"Kind", report.PolicyReport.Kind,
		)
		os.Exit(1)
	}

	// Create a cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go metrics.Start(ctx)

	if ns, ok := os.LookupEnv(env.LeaderElectionNS); ok && strings.TrimSpace(ns) != "" {
		if err := handler.RunAsLeader(ctx, cancel, ns, run); err != nil {
			slog.Error("error running with leader election", "error", err)
			os.Exit(1)
		}
	} else {
		run(ctx, handler, cancel)
	}
}

func run(ctx context.Context, handler report.Handler, cancel context.CancelFunc) {
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

	start(ctx, reportChan, cancel, "KubeArmor", env.KubeArmorServiceName, kubearmor.Run)
	start(ctx, reportChan, cancel, "Hubble", env.HubbleServiceName, hubble.Run)

	// Process reports from the channel
	for {
		select {
		case rep, ok := <-reportChan:
			if !ok {
				// Channel closed, exit loop
				return
			}
			if err := handler.Update(ctx, rep); err != nil {
				slog.Error("Failed to update report", "error", err)
			}
		case <-ctx.Done():
			// Context is done, exit loop
			slog.Info("Context done, exiting report processing loop.")
			return
		}
	}
}

func start(ctx context.Context,
	reportChan chan *report.Item,
	cancel context.CancelFunc,
	name string,
	serviceVar string,
	run func(ctx context.Context, reportChan chan *report.Item) error,
) {
	if !env.Empty(serviceVar) {
		go func() {
			slog.Info("starting", "name", name, "service", os.Getenv(serviceVar))
			if err := run(ctx, reportChan); err != nil {
				slog.Error("run exited with error", "name", name, "error", err)
				cancel()
			}
		}()
	}
}
