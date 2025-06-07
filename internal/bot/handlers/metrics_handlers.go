package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hackaton/internal/monitoring"

	telebot "gopkg.in/telebot.v3"
)

type Handler struct {
	monitor *monitoring.Client
}

func New(monitor *monitoring.Client) *Handler {
	return &Handler{monitor: monitor}
}

// metric
func (h *Handler) MetricHandler(c telebot.Context) error {
	parts := strings.SplitN(c.Text(), " ", 3)
	if len(parts) < 3 {
		return c.Send("Неправильное кол-во параметров ")
	}
	service := parts[1]
	metric := parts[2]
	req := metric + "{job=\"" + service + "\"}"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response, err := h.monitor.Query(ctx, req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return c.Send("Превышено время ожидания запроса (timeout)")
		}
		return c.Send(fmt.Sprintf("Произошла ошибка: %v", err))
	}

	result := fmt.Sprintf("Status: %s\n", response.Status)

	// TODO: че блять
	for i, res := range response.Data.Result {
		result += fmt.Sprintf("data.result[%d].value: %v\n", i, res.Value)
	}

	return c.Send(result)

}

// metric
func (h *Handler) ListMetricsHandler(c telebot.Context) error {
	parts := strings.SplitN(c.Text(), " ", 3)
	if len(parts) < 3 {
		return c.Send("Неправильное кол-во параметров ")
	}
	service := parts[1]
	req := service
	metric := parts[2]
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response, err := h.monitor.ListMetrics(ctx, req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return c.Send("Превышено время ожидания запроса (timeout)")
		}
		return c.Send(fmt.Sprintf("Произошла ошибка: %v", err))
	}
	var matchedMetrics []string
	for _, str := range response {
		if strings.Contains(str, metric) {
			matchedMetrics = append(matchedMetrics, str)
		}
	}

	return c.Send(strings.Join(matchedMetrics, "\n"))
}

// StatusHandler - заглушка
func (h *Handler) StatusHandler(c telebot.Context) error {
	return c.Send("Not implemented")
}

// ScaleHandler - заглушка
func (h *Handler) ScaleHandler(c telebot.Context) error {
	return c.Send("Not implemented")
}

// RestartHandler - заглушка
func (h *Handler) RestartHandler(c telebot.Context) error {
	return c.Send("Not implemented")
}

// RollbackHandler - заглушка
func (h *Handler) RollbackHandler(c telebot.Context) error {
	return c.Send("Not implemented")
}

// HistoryHandler - заглушка
func (h *Handler) HistoryHandler(c telebot.Context) error {
	return c.Send("Not implemented")
}

// OperationsHandler - заглушка
func (h *Handler) OperationsHandler(c telebot.Context) error {
	return c.Send("Not implemented")
}
