package kube

import "context"

type Client struct {
	// TODO: добавить clientset (*kubernetes.Clientset)
}

func NewClient() (*Client, error) {
	// TODO: настроить подключение к кластеру
	return &Client{}, nil
}

func (c *Client) GetStatus(ctx context.Context, serviceNameOrAlias string) (string, error) {
	// TODO: реализовать поиск по имени и по лейблу
	return "Status: OK", nil
} 