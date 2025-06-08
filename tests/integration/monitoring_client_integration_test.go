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
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Println("Error loading .env file, assuming env vars are set externally")
	}
	os.Exit(m.Run())
}

func TestMonitoringClient_Integration(t *testing.T) {
	promURLRaw := os.Getenv("PROMETHEUS_URL")
	alertmanagerURL := os.Getenv("ALERTMANAGER_URL")
	user := os.Getenv("PROMETHEUS_USER")
	pass := os.Getenv("PROMETHEUS_PASS")
	jobName := os.Getenv("TEST_JOB_NAME")
	namespace := os.Getenv("TEST_NAMESPACE")

	if promURLRaw == "" || alertmanagerURL == "" || user == "" || pass == "" || jobName == "" {
		t.Skip("Skipping integration test: required environment variables are not set")
	}

	u, err := url.Parse(promURLRaw)
	require.NoError(t, err)
	u.User = url.UserPassword(user, pass)
	promURL := u.String()

	client, err := monitoring.NewClient(promURL, alertmanagerURL)
	require.NoError(t, err, "Failed to create monitoring client")

	ctx := context.Background()

	t.Run("ListMetrics", func(t *testing.T) {
		metricNames, err := client.ListMetrics(ctx, jobName)
		if err != nil {
			t.Fatalf("Failed to list metrics: %v", err)
		}

		if len(metricNames) == 0 {
			t.Errorf("Expected to get some metrics, but got none")
		}

		fmt.Printf("Successfully listed %d metrics for job %s\n", len(metricNames), jobName)
	})

	t.Run("QueryMetric", func(t *testing.T) {
		metricToQuery := "go_goroutines"
		query := fmt.Sprintf(`%s{job=~".*%s.*", namespace="%s"}`, metricToQuery, jobName, namespace)
		queryResp, err := client.Query(ctx, query)
		if err != nil {
			t.Fatalf("Failed to query metric: %v", err)
		}

		if len(queryResp.Data.Result) == 0 {
			t.Errorf("No data found for metric %s", metricToQuery)
			return
		}

		fmt.Printf("Successfully queried metric '%s'. Found %d time series.\n", metricToQuery, len(queryResp.Data.Result))
		for _, res := range queryResp.Data.Result {
			podName := res.Metric["pod"]
			value := res.Value[1]
			fmt.Printf("  - Pod: %s, Value: %v\n", podName, value)
		}
	})
}
