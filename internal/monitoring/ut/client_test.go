package monitoring_test

import (
	"context"
	"encoding/json"
	"fmt"
	"hackaton/internal/monitoring"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func setupTestClient(t *testing.T, handler http.Handler) (*monitoring.Client, *httptest.Server) {
	server := httptest.NewServer(handler)
	client, err := monitoring.NewClient(server.URL, "")
	if err != nil {
		server.Close()
		t.Fatalf("Failed to create test client: %v", err)
	}
	return client, server
}

func TestClient_ListMetrics(t *testing.T) {
	expectedMetrics := []string{"metric1", "metric2", "go_goroutines"}
	jobName := "test-job"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/label/__name__/values" {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		queryParam := r.URL.Query().Get("match[]")
		expectedQuery := fmt.Sprintf("{job=%q}", jobName)
		if queryParam != expectedQuery {
			http.Error(w, fmt.Sprintf("Bad match[] param: got %s, want %s", queryParam, expectedQuery), http.StatusBadRequest)
			return
		}

		resp := monitoring.PrometheusLabelResponse{
			Status: "success",
			Data:   expectedMetrics,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	client, server := setupTestClient(t, handler)
	defer server.Close()

	metrics, err := client.ListMetrics(context.Background(), jobName)
	if err != nil {
		t.Fatalf("ListMetrics() returned an error: %v", err)
	}

	if !reflect.DeepEqual(metrics, expectedMetrics) {
		t.Errorf("ListMetrics() got = %v, want %v", metrics, expectedMetrics)
	}
}

func TestClient_Query(t *testing.T) {
	expectedQuery := `go_goroutines{job="test-job"}`

	mockResponseJSON := `{
		"status": "success",
		"data": {
			"resultType": "vector",
			"result": [
				{
					"metric": {
						"pod": "pod-1"
					},
					"value": [
						12345.0,
						"9"
					]
				}
			]
		}
	}`

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/query" {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("query") != expectedQuery {
			http.Error(w, "Bad query param", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResponseJSON))
	})

	client, server := setupTestClient(t, handler)
	defer server.Close()

	resp, err := client.Query(context.Background(), expectedQuery)
	if err != nil {
		t.Fatalf("Query() returned an error: %v", err)
	}

	if resp.Status != "success" {
		t.Errorf("Query() response status got = %s, want success", resp.Status)
	}

	if len(resp.Data.Result) != 1 {
		t.Fatalf("Query() expected 1 result, got %d", len(resp.Data.Result))
	}

	if resp.Data.Result[0].Value[1] != "9" {
		t.Errorf("Query() result value got = %v, want 9", resp.Data.Result[0].Value[1])
	}
}
