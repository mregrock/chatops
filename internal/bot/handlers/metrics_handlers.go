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
func MetricHandler(c telebot.Context) error {
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

/**
 * получить статус подов
 */
func StatusHandler(c telebot.Context) error {
	// return c.Send("Not implemented")

	parts := strings.SplitN(c.Text(), " ", 2)
	if len(parts) < 2 {
		return c.Send("Неправильное кол-во параметров ")
	}
	job := parts[1]
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response, err := GlobalMonitorClient.GetStatusDashboard(ctx, "", job)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return c.Send("Превышено время ожидания запроса (timeout)")
		}
		return c.Send(fmt.Sprintf("Произошла ошибка: %v", err))
	}

	return c.Send(FormatDashboardForTelegram(response), "\n")
}

// FormatDashboardForTelegram форматирует данные дашборда в строку для отправки в Telegram
func FormatDashboardForTelegram(dashboard *monitoring.ServiceStatusDashboard) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("*Статус сервиса: `%s`*\n\n", escapeMarkdown(dashboard.ServiceName)))

	if len(dashboard.Alerts) > 0 {
		sb.WriteString("🔥 *Активные алерты:*\n")
		for _, alert := range dashboard.Alerts {
			// Using block quotes for alerts
			sb.WriteString(fmt.Sprintf("> %s\n", escapeMarkdown(alert.Labels["alertname"])))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("✅ *Нет активных алертов*\n\n")
	}

	if len(dashboard.Pods) > 0 {
		sb.WriteString("💻 *Pods:*\n")
		for _, pod := range dashboard.Pods {
			sb.WriteString("--------------------------------\n")
			readyIcon := "✅"
			if !pod.Ready {
				readyIcon = "⏳"
			}

			// Use human-readable memory units
			memUsageMiB := pod.MemoryUsageBytes / 1024 / 1024
			memLimitMiB := pod.MemoryLimitBytes / 1024 / 1024

			sb.WriteString(fmt.Sprintf("*Pod:* `%s`\n", escapeMarkdown(pod.PodName)))
			sb.WriteString(fmt.Sprintf("*Status:* %s %s\n", readyIcon, escapeMarkdown(pod.Phase)))
			sb.WriteString(fmt.Sprintf("*CPU:* `%.2f / %.2f` cores\n", pod.CPUUsageCores, pod.CPULimitCores))
			sb.WriteString(fmt.Sprintf("*Memory:* `%.0f / %.0f` MiB\n", memUsageMiB, memLimitMiB))
			sb.WriteString(fmt.Sprintf("*Restarts:* `%d`\n", pod.Restarts))
			if pod.OOMKilled {
				sb.WriteString("*OOMKilled:* 💀 `true`\n")
			}
		}
	} else {
		sb.WriteString("🤷‍♂️ *Подов по запросу не найдено\\.*\n")
	}

	return sb.String()
}

// escapeMarkdown escapes characters that have special meaning in Telegram's MarkdownV2.
func escapeMarkdown(s string) string {
	r := strings.NewReplacer(
		"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]", "(",
		"\\(", ")", "\\)", "~", "\\~", "`", "\\`", ">",
		"\\>", "#", "\\#", "+", "\\+", "-", "\\-", "=",
		"\\=", "|", "\\|", "{", "\\{", "}", "\\}", ".",
		"\\.", "!", "\\!",
	)
	return r.Replace(s)
}
