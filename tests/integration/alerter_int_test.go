//go:build integration

package integration

import (
	"bytes"
	"context"
	"hackaton/internal/app"
	"hackaton/internal/monitoring"
	"log"
	"os"
	"strings"
	"testing"

	"db/config"
	"db/repository"
)

func TestAlerter_Integration_CheckAndNotify(t *testing.T) {
	log.Println("Connecting to DB for integration test...")
	if err := config.InitDB(); err != nil {
		t.Fatalf("Failed to connect to DB: %v", err)
	}

	config.DB.Exec("DELETE FROM user_labels")
	config.DB.Exec("DELETE FROM users")

	testUser, err := repository.CreateUser("duty_officer_test", "password", "on-call")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}
	if err := repository.UpdateUserDutyStatus(testUser.ID, true); err != nil {
		t.Fatalf("Failed to set user duty status: %v", err)
	}

	const testLabel = "severity=critical"
	if _, err := repository.CreateUserLabel(testUser.ID, testLabel); err != nil {
		t.Fatalf("Failed to assign label to user: %v", err)
	}
	log.Printf("Test user '%s' created and assigned to label '%s'", testUser.Login, testLabel)

	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	t.Cleanup(func() {
		log.SetOutput(os.Stderr)
	})

	alertmanagerURL := "http://localhost:9093"
	monClient, err := monitoring.NewClient("", alertmanagerURL)
	if err != nil {
		t.Fatalf("Failed to create monitoring client: %v", err)
	}

	dbAdapter := &app.DBAdapter{}
	alerter := app.NewAlerter(monClient, dbAdapter)

	log.Println("Running integration test for Alerter. Checking for alerts...")
	err = alerter.CheckAndNotify(context.Background())
	if err != nil {
		t.Fatalf("Alerter's CheckAndNotify failed: %v", err)
	}

	logOutput := logBuffer.String()

	expectedNotification := "УВЕДОМЛЕНИЕ ДЛЯ: @" + testUser.Login
	if !strings.Contains(logOutput, expectedNotification) {
		t.Errorf("Expected log to contain notification for '%s', but it was not found.", testUser.Login)
	}

	expectedAlertName := "KubeProxyDown"
	if !strings.Contains(logOutput, expectedAlertName) {
		t.Errorf("Expected notification to be for alert '%s', but it was not found.", expectedAlertName)
	}

	log.SetOutput(os.Stderr)
	log.Println("Integration test finished successfully. Notification found in logs.")
}
