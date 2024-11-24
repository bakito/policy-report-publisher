package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bakito/policy-report-publisher/pkg/hubble"
	"github.com/bakito/policy-report-publisher/pkg/kubearmor"
	"github.com/bakito/policy-report-publisher/pkg/report"
	"github.com/bakito/policy-report-publisher/version"
)

func main() {
	var withHubble bool
	var withKubeArmor bool

	flag.BoolVar(&withHubble, "hubble", false, "enable hubble")
	flag.BoolVar(&withKubeArmor, "kubearmor", false, "enable kubearmor")
	flag.Parse()

	if !withKubeArmor && !withHubble {
		log.Fatalf("either 'hubble' or 'kubearmor' must be enabled")
	}

	log.Printf("policy-report-publisher %s", version.Version)

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
	}()

	if withKubeArmor {
		go func() {
			if err := kubearmor.Run(ctx, reportChan); err != nil {
				log.Printf("kubearmor.Run exited with error: %v", err)
			}
		}()
	}
	if withHubble {
		go func() {
			if err := hubble.Run(ctx, reportChan); err != nil {
				log.Printf("hubble.Run exited with error: %v", err)
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
				log.Printf("Failed to update report: %v", err)
			}
		case <-ctx.Done():
			// Context is done, exit loop
			log.Println("Context done, exiting report processing loop.")
			return
		}
	}
}
