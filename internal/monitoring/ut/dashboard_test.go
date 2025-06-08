package monitoring_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"chatops/internal/monitoring"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStatusDashboard_Integration(t *testing.T) {
	promServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/query" {
			http.NotFound(w, r)
			return
		}

		query := r.URL.Query().Get("query")
		var resp monitoring.PrometheusQueryResponse
		resp.Status = "success"
		resp.Data.ResultType = "vector"

		switch query {
		case `sum(rate(container_cpu_usage_seconds_total{job="test-job", namespace="test-ns", container!=""}[5m])) by (pod)`:
			resp.Data.Result = []struct {
				Metric map[string]string `json:"metric"`
				Value  []interface{}     `json:"value"`
			}{
				{Metric: map[string]string{"pod": "pod-1"}, Value: []interface{}{1672531200.0, "0.5"}},
				{Metric: map[string]string{"pod": "pod-2"}, Value: []interface{}{1672531200.0, "1.2"}},
			}
		case `sum(kube_pod_container_resource_limits{job="test-job", namespace="test-ns", resource="cpu"}) by (pod)`:
			resp.Data.Result = []struct {
				Metric map[string]string `json:"metric"`
				Value  []interface{}     `json:"value"`
			}{
				{Metric: map[string]string{"pod": "pod-1"}, Value: []interface{}{1672531200.0, "1"}},
				{Metric: map[string]string{"pod": "pod-2"}, Value: []interface{}{1672531200.0, "2"}},
			}
		case `sum(container_memory_working_set_bytes{job="test-job", namespace="test-ns", container!=""}) by (pod)`:
			resp.Data.Result = []struct {
				Metric map[string]string `json:"metric"`
				Value  []interface{}     `json:"value"`
			}{
				{Metric: map[string]string{"pod": "pod-1"}, Value: []interface{}{1672531200.0, "1073741824"}}, // 1 GiB
			}
		case `sum(kube_pod_container_resource_limits{job="test-job", namespace="test-ns", resource="memory"}) by (pod)`:
			resp.Data.Result = []struct {
				Metric map[string]string `json:"metric"`
				Value  []interface{}     `json:"value"`
			}{
				{Metric: map[string]string{"pod": "pod-1"}, Value: []interface{}{1672531200.0, "2147483648"}}, // 2 GiB
			}
		case `sum(kube_pod_container_status_restarts_total{job="test-job", namespace="test-ns"}) by (pod)`:
			resp.Data.Result = []struct {
				Metric map[string]string `json:"metric"`
				Value  []interface{}     `json:"value"`
			}{
				{Metric: map[string]string{"pod": "pod-2"}, Value: []interface{}{1672531200.0, "5"}},
			}
		case `kube_pod_container_status_last_terminated_reason{job="test-job", namespace="test-ns", reason="OOMKilled"}`:
			resp.Data.Result = []struct {
				Metric map[string]string `json:"metric"`
				Value  []interface{}     `json:"value"`
			}{
				{Metric: map[string]string{"pod": "pod-2", "namespace": "test-ns", "reason": "OOMKilled"}, Value: []interface{}{1672531200.0, "1"}},
			}
		default:
			resp.Data.Result = []struct {
				Metric map[string]string `json:"metric"`
				Value  []interface{}     `json:"value"`
			}{}
		}

		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(resp)
		if err != nil {
			t.Fatalf("Failed to encode mock prometheus response: %v", err)
		}
	}))
	defer promServer.Close()

	alertmanagerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/alerts" {
			http.NotFound(w, r)
			return
		}
		alerts := []monitoring.Alert{
			{
				Labels:      map[string]string{"alertname": "HighCPU", "job": "test-job", "namespace": "test-ns"},
				Annotations: map[string]string{"summary": "CPU usage is high"},
				State:       "firing",
				ActiveAt:    time.Now(),
				Value:       "95",
			},
			{
				Labels:      map[string]string{"alertname": "OtherAlert", "job": "other-job"}, // Should be filtered out
				Annotations: map[string]string{"summary": "Another alert"},
				State:       "firing",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(alerts)
		if err != nil {
			t.Fatalf("Failed to encode mock alertmanager response: %v", err)
		}
	}))
	defer alertmanagerServer.Close()

	client, err := monitoring.NewClient(promServer.URL, alertmanagerServer.URL)
	require.NoError(t, err)

	dashboard, err := client.GetStatusDashboard(context.Background(), "test-ns", "test-job")
	require.NoError(t, err)
	require.NotNil(t, dashboard)

	assert.Equal(t, "test-job", dashboard.ServiceName)

	require.Len(t, dashboard.Alerts, 1)
	assert.Equal(t, "HighCPU", dashboard.Alerts[0].Labels["alertname"])

	require.Len(t, dashboard.Pods, 2)

	sort.Slice(dashboard.Pods, func(i, j int) bool {
		return dashboard.Pods[i].PodName < dashboard.Pods[j].PodName
	})

	pod1 := dashboard.Pods[0]
	assert.Equal(t, "pod-1", pod1.PodName)
	assert.Equal(t, 0.5, pod1.CPUUsageCores)
	assert.Equal(t, 1.0, pod1.CPULimitCores)
	assert.Equal(t, 1073741824.0, pod1.MemoryUsageBytes)
	assert.Equal(t, 2147483648.0, pod1.MemoryLimitBytes)
	assert.Equal(t, int64(0), pod1.Restarts)
	assert.False(t, pod1.OOMKilled)

	pod2 := dashboard.Pods[1]
	assert.Equal(t, "pod-2", pod2.PodName)
	assert.Equal(t, 1.2, pod2.CPUUsageCores)
	assert.Equal(t, 2.0, pod2.CPULimitCores)
	assert.Equal(t, 0.0, pod2.MemoryUsageBytes)
	assert.Equal(t, 0.0, pod2.MemoryLimitBytes)
	assert.Equal(t, int64(5), pod2.Restarts)
	assert.True(t, pod2.OOMKilled)
}

func TestGetStatusDashboard_ErrorHandling(t *testing.T) {
	t.Run("prometheus unavailable", func(t *testing.T) {
		alertmanagerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `[]`)
		}))
		defer alertmanagerServer.Close()

		invalidPromURL := "http://localhost:9999"

		client, err := monitoring.NewClient(invalidPromURL, alertmanagerServer.URL)
		require.NoError(t, err)

		dashboard, err := client.GetStatusDashboard(context.Background(), "test-ns", "test-job")
		require.NoError(t, err)
		require.NotNil(t, dashboard)

		assert.Empty(t, dashboard.Pods)
		assert.Empty(t, dashboard.Alerts)
	})

	t.Run("alertmanager unavailable", func(t *testing.T) {
		promServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `{"status":"success","data":{"resultType":"vector","result":[]}}`)
		}))
		defer promServer.Close()

		invalidAlertmanagerURL := "http://localhost:9998"

		client, err := monitoring.NewClient(promServer.URL, invalidAlertmanagerURL)
		require.NoError(t, err)

		dashboard, err := client.GetStatusDashboard(context.Background(), "test-ns", "test-job")
		require.NoError(t, err)
		require.NotNil(t, dashboard)

		assert.Empty(t, dashboard.Alerts)
		assert.Empty(t, dashboard.Pods)
	})
}
