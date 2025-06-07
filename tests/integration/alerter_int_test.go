//go:build integration

package integration

import (
	"context"
	"hackaton/internal/app"
	"hackaton/internal/monitoring"
	"log"
	"os"
	"testing"
)

func TestAlerter_Integration_CheckAndNotify(t *testing.T) {
	alertmanagerURL := os.Getenv("ALERTMANAGER_URL")
	if alertmanagerURL == "" {
		alertmanagerURL = "http://localhost:9093"
	}

	log.Printf("Using Alertmanager URL: %s", alertmanagerURL)

	promURL := ""
	monClient, err := monitoring.NewClient(promURL, alertmanagerURL)
	if err != nil {
		t.Fatalf("Failed to create monitoring client: %v", err)
	}

	alerter := app.NewAlerter(monClient)

	log.Println("Running integration test for Alerter. Checking for alerts...")
	err = alerter.CheckAndNotify(context.Background())
	if err != nil {
		t.Fatalf("Alerter's CheckAndNotify failed: %v", err)
	}

	log.Println("Integration test finished. Check logs for notification details.")
}
