package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"chatops/internal/monitoring"

	telebot "gopkg.in/telebot.v3"
)

var GlobalMonitorClient *monitoring.Client 

// SetMonitorClient sets the global monitor client for handlers
func SetMonitorClient(client *monitoring.Client) {
	GlobalMonitorClient = client
}

// metric
func  MetricHandler(c telebot.Context) error {
	parts := strings.SplitN(c.Text(), " ", 3)
	if len(parts) < 3 {
		return c.Send("Неправильное кол-во параметров ")
	}
	service := parts[1]
	metric := parts[2]
	req := metric + "{job=\"" + service + "\"}"
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response, err := GlobalMonitorClient.Query(ctx, req)


	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return c.Send("Превышено время ожидания запроса (timeout)")
		}
		return c.Send(fmt.Sprintf("Произошла ошибка: %v", err))
	}

	result := fmt.Sprintf("Status: %s\n", response.Status)

	var allValues string

	for _, result := range response.Data.Result {
		allValues += fmt.Sprintf("%v: ", result.Metric["pod"])
		for _, v := range result.Value {
			allValues += fmt.Sprintf("%v ", v)
		}
	}

	allValues = strings.TrimSpace(allValues) 

	return c.Send(result + allValues)

}

// metric
func ListMetricsHandler(c telebot.Context) error {
	parts := strings.SplitN(c.Text(), " ", 3)
	if len(parts) < 3 {
		return c.Send("Неправильное кол-во параметров ")
	}
	service := parts[1]
	req := service
	metric := parts[2]
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response, err := GlobalMonitorClient.ListMetrics(ctx, req)
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
func StatusHandler(c telebot.Context) error {
	return c.Send("Not implemented")
}

// ScaleHandler - заглушка
func ScaleHandler(c telebot.Context) error {
	return c.Send("Not implemented")
}

// RestartHandler - заглушка
func  RestartHandler(c telebot.Context) error {
	return c.Send("Not implemented")
}

// RollbackHandler - заглушка
func  RollbackHandler(c telebot.Context) error {
	return c.Send("Not implemented")
}



// OperationsHandler - заглушка
// func (h *Handler) OperationsHandler(c telebot.Context) error {
// 	return c.Send("Not implemented")
// }


func RevisionsHandler(c telebot.Context) error {
	// TODO: Реализовать логику для команды revisions
	return c.Send("Выполняется команда revisions...")
}