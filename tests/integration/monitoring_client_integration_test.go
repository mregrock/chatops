//go:build integration
// +build integration

package integration

import (
	"context"
	"fmt"
	"hackaton/internal/monitoring"
	"testing"
)

func TestMonitoringClient_Integration(t *testing.T) {
	promURL := "http://localhost:9090"
	alertmanagerURL := "http://localhost:9093"

	client, err := monitoring.NewClient(promURL, alertmanagerURL)
	if err != nil {
		t.Fatalf("Failed to create monitoring client: %v", err)
	}

	ctx := context.Background()
	jobName := "test-app-go-svc"

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
		query := fmt.Sprintf(`%s{job=%q}`, metricToQuery, jobName)
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
