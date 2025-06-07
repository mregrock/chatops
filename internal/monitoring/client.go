package monitoring

import "context"

type Client struct {
	// TODO: добавить HTTP клиент и URL'ы к Prometheus/Alertmanager
}

// NewClient создает нового клиента.
func NewClient() (*Client, error) {
	return &Client{}, nil
}

func (c *Client) GetActiveAlerts(ctx context.Context) ([]string, error) {
	// TODO: выполнить HTTP GET запрос к Alertmanager API
	return []string{"Host down: server1"}, nil
} 