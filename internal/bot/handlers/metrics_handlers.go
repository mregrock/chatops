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
	parts := strings.Split(c.Text(), " ")
	if len(parts) < 3 {
		return c.Send("Использование: /metric <сервис> <метрика> [namespace]")
	}
	service := parts[1]
	metric := parts[2]
	namespace := "default"
	if len(parts) > 3 {
		namespace = parts[3]
	}

	req := fmt.Sprintf(`%s{job=~"^%s.*", namespace="%s"}`, metric, service, namespace)
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

    fmt.Println("Getting metrics...")
	response, err := GlobalMonitorClient.ListMetrics(ctx, req)
	fmt.Println("Resp metrics:")
	fmt.Println(response)
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

	parts := strings.Split(c.Text(), " ")

	if len(parts) < 2 {
		return c.Send("Usage: /status <job_name> [namespace]")
	}
	job := parts[1]
	namespace := "default"
	if len(parts) > 2 {
		namespace = parts[2]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	fmt.Println("Getting status dashboard for job:", job, "in namespace:", namespace)
	response, err := GlobalMonitorClient.GetStatusDashboard(ctx, namespace, job)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return c.Send("Превышено время ожидания запроса (timeout)")
		}
		return c.Send(fmt.Sprintf("❌ *Произошла ошибка:*\n`%v`", escapeMarkdown(err.Error())), telebot.ModeMarkdownV2)
	}

	fmt.Printf("Successfully got dashboard: %+v\n", response)

	return c.Send(FormatDashboardForTelegram(response), telebot.ModeMarkdownV2)

}

// FormatDashboardForTelegram форматирует данные дашборда в строку для отправки в Telegram
func FormatDashboardForTelegram(dashboard *monitoring.ServiceStatusDashboard) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("*Статус сервиса: `%s`*\n\n", escapeMarkdown(dashboard.ServiceName)))

	if len(dashboard.Alerts) > 0 {
		sb.WriteString("🔥 *Активные алерты:*\n")
		for _, alert := range dashboard.Alerts {
			alertName := alert.Labels["alertname"]
			summary := alert.Annotations["summary"]
			if summary == "" {
				summary = "Нет описания."
			}
			// Use block quotes for alerts for better visibility
			sb.WriteString(fmt.Sprintf("> *%s:* %s\n", escapeMarkdown(alertName), escapeMarkdown(summary)))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("✅ *Нет активных алертов*\n\n")
	}

	if len(dashboard.Pods) > 0 {
		sb.WriteString("💻 *Поды:*\n")
		for _, pod := range dashboard.Pods {
			sb.WriteString("================================\n")

			var statusIcon, statusText string
			switch {
			case pod.Ready:
				statusIcon = "✅"
				statusText = "Ready"
			case pod.Phase == "Running" && !pod.Ready:
				statusIcon = "⏳"
				statusText = "Not Ready"
			case pod.Phase == "Pending":
				statusIcon = "⌛️"
				statusText = "Pending"
			case pod.Phase == "Succeeded":
				statusIcon = "🏁"
				statusText = "Succeeded"
			default:
				statusIcon = "❌"
				statusText = pod.Phase
			}

			memUsageMiB := pod.MemoryUsageBytes / 1024 / 1024
			memLimitMiB := pod.MemoryLimitBytes / 1024 / 1024

			sb.WriteString(fmt.Sprintf("*Под:* `%s`\n", escapeMarkdown(pod.PodName)))
			sb.WriteString(fmt.Sprintf("*Статус:* %s %s\n", statusIcon, escapeMarkdown(statusText)))
			sb.WriteString(fmt.Sprintf("*CPU:* `%.2f / %.2f` cores\n", pod.CPUUsageCores, pod.CPULimitCores))
			sb.WriteString(fmt.Sprintf("*Память:* `%.0f / %.0f` MiB\n", memUsageMiB, memLimitMiB))
			sb.WriteString(fmt.Sprintf("*Перезапуски:* `%d`\n", pod.Restarts))
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
