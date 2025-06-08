package monitoring

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type PrometheusQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

type PrometheusLabelResponse struct {
	Status string   `json:"status"`
	Data   []string `json:"data"`
}

type Alert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	State       string            `json:"state"`
	ActiveAt    time.Time         `json:"activeAt"`
	Value       string            `json:"value"`
}

type Client struct {
	httpClient      *http.Client
	prometheusURL   string
	alertmanagerURL string
}

func NewClient(prometheusURL, alertmanagerURL string) (*Client, error) {
	if prometheusURL != "" {
		if _, err := url.ParseRequestURI(prometheusURL); err != nil {
			return nil, fmt.Errorf("invalid prometheus URL: %w", err)
		}
	}

	if alertmanagerURL != "" {
		if _, err := url.ParseRequestURI(alertmanagerURL); err != nil {
			return nil, fmt.Errorf("invalid alertmanager URL: %w", err)
		}
	}

	// Создаем транспорт, который игнорирует проверку TLS-сертификата
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	return &Client{
		prometheusURL:   prometheusURL,
		alertmanagerURL: alertmanagerURL,
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: tr, // Используем кастомный транспорт
		},
	}, nil
}

func (c *Client) GetActiveAlerts(ctx context.Context) ([]Alert, error) {
	endpoint := fmt.Sprintf("%s/api/v2/alerts", c.alertmanagerURL)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request to alertmanager: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("alertmanager returned non-OK status: %s", resp.Status)
	}

	var alerts []Alert
	if err := json.NewDecoder(resp.Body).Decode(&alerts); err != nil {
		return nil, fmt.Errorf("failed to decode alertmanager response: %w", err)
	}

	return alerts, nil
}

func (c *Client) ListMetrics(ctx context.Context, jobName string) ([]string, error) {
	endpoint := fmt.Sprintf("%s/api/v1/label/__name__/values", c.prometheusURL)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("match[]", fmt.Sprintf("{job=%q}", jobName))
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prometheus returned non-OK status: %s", resp.Status)
	}

	var promResp PrometheusLabelResponse
	if err := json.NewDecoder(resp.Body).Decode(&promResp); err != nil {
		return nil, fmt.Errorf("failed to decode prometheus response: %w", err)
	}

	if promResp.Status != "success" {
		return nil, fmt.Errorf("prometheus returned non-success status: %s", promResp.Status)
	}

	return promResp.Data, nil
}

func (c *Client) Query(ctx context.Context, query string) (*PrometheusQueryResponse, error) {
	endpoint := fmt.Sprintf("%s/api/v1/query", c.prometheusURL)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Add("query", query)
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prometheus returned non-OK status: %s", resp.Status)
	}

	var promResp PrometheusQueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&promResp); err != nil {
		return nil, fmt.Errorf("failed to decode prometheus response: %w", err)
	}

	if promResp.Status != "success" {
		return nil, fmt.Errorf("prometheus returned non-success status: %s", promResp.Status)
	}

	return &promResp, nil
}
