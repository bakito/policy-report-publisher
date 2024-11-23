package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bakito/policy-reporter-plugin/pkg/hubble"
	"github.com/bakito/policy-reporter-plugin/pkg/kubearmor"
	"github.com/bakito/policy-reporter-plugin/pkg/report"
)

func main() {
	// Initialize the report handler
	handler, err := report.NewHandler()
	if err != nil {
		log.Fatalf("Failed to create report handler: %v", err)
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
		log.Println("Shutting down gracefully...")
		cancel()
		close(reportChan)
	}()

	// Start kubearmor and hubble as producers
	go func() {
		if err := kubearmor.Run(ctx, reportChan); err != nil {
			log.Printf("kubearmor.Run exited with error: %v", err)
		}
	}()
	go func() {
		if err := hubble.Run(ctx, reportChan); err != nil {
			log.Printf("hubble.Run exited with error: %v", err)
		}
	}()

	// Process reports from the channel
	for report := range reportChan {
		if err := handler.Update(ctx, report); err != nil {
			log.Printf("Failed to update report: %v", err)
		}
	}
}
