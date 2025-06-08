//go:build integration
// +build integration

package integration

import (
	"chatops/internal/monitoring"
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Println("Error loading .env file, assuming env vars are set externally")
	}
	os.Exit(m.Run())
}

func TestGetStatusDashboard_Integration(t *testing.T) {
	promURLRaw := os.Getenv("PROMETHEUS_URL")
	alertmanagerURL := os.Getenv("ALERTMANAGER_URL")
	user := os.Getenv("PROMETHEUS_USER")
	pass := os.Getenv("PROMETHEUS_PASS")
	jobName := os.Getenv("TEST_JOB_NAME")
	namespace := os.Getenv("TEST_NAMESPACE")

	if promURLRaw == "" || alertmanagerURL == "" || user == "" || pass == "" || jobName == "" {
		t.Skip("Skipping integration test: required environment variables are not set (PROMETHEUS_URL, ALERTMANAGER_URL, PROMETHEUS_USER, PROMETHEUS_PASS, TEST_JOB_NAME)")
	}

	u, err := url.Parse(promURLRaw)
	require.NoError(t, err)
	u.User = url.UserPassword(user, pass)
	promURL := u.String()

	client, err := monitoring.NewClient(promURL, alertmanagerURL)
	require.NoError(t, err, "Failed to create monitoring client")

	ctx := context.Background()

	t.Run("GetDashboardForKnownService", func(t *testing.T) {
		dashboard, err := client.GetStatusDashboard(ctx, namespace, jobName)
		require.NoError(t, err, "GetStatusDashboard returned an error")
		require.NotNil(t, dashboard, "Dashboard should not be nil")

		fmt.Printf("--- Dashboard for Service: %s ---\n", dashboard.ServiceName)
		fmt.Printf("Found %d active alerts.\n", len(dashboard.Alerts))
		for _, alert := range dashboard.Alerts {
			fmt.Printf("  - ALERT: %s, Summary: %s\n", alert.Labels["alertname"], alert.Annotations["summary"])
		}

		fmt.Printf("Found %d pods.\n", len(dashboard.Pods))
		if len(dashboard.Pods) == 0 {
			t.Log("Warning: no pods found for this service. The service might be down or has no running pods.")
		}

		for _, pod := range dashboard.Pods {
			fmt.Printf("  - POD: %s\n", pod.PodName)
			fmt.Printf("    Status: %s (Ready: %v)\n", pod.Phase, pod.Ready)
			fmt.Printf("    CPU Usage: %.4f / %.4f Cores\n", pod.CPUUsageCores, pod.CPULimitCores)
			fmt.Printf("    Memory Usage: %.2f / %.2f MiB\n", pod.MemoryUsageBytes/1024/1024, pod.MemoryLimitBytes/1024/1024)
			fmt.Printf("    Restarts: %d\n", pod.Restarts)
			fmt.Printf("    OOMKilled: %v\n", pod.OOMKilled)

			assert.Equal(t, jobName, dashboard.ServiceName)
		}
		fmt.Println("--- End of Dashboard ---")
	})

	t.Run("GetDashboardForNonExistentService", func(t *testing.T) {
		nonExistentJob := "i-do-not-exist-for-real"
		dashboard, err := client.GetStatusDashboard(ctx, namespace, nonExistentJob)
		require.NoError(t, err, "GetStatusDashboard should not return an error for a non-existent service")
		require.NotNil(t, dashboard)

		assert.Equal(t, nonExistentJob, dashboard.ServiceName)
		assert.Empty(t, dashboard.Alerts, "Should be no alerts for a non-existent service")
		assert.Empty(t, dashboard.Pods, "Should be no pods for a non-existent service")

		fmt.Printf("\nSuccessfully verified that dashboard is empty for non-existent service '%s'.\n", nonExistentJob)
	})
}
