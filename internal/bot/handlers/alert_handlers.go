package handlers

import (
	"context"
	"fmt"
	"strings"
	"time"

	"chatops/internal/monitoring"

	telebot "gopkg.in/telebot.v3"
)

func AlertsHandler(c telebot.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	message, err := GenerateAlertsMessage(ctx, GlobalMonitorClient)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return c.Send("Превышено время ожидания запроса (timeout)")
		}
		return c.Send(fmt.Sprintf("❌ *Произошла ошибка:*\n`%v`", escapeMarkdown(err.Error())), telebot.ModeMarkdownV2)
	}

	return c.Send(message, telebot.ModeMarkdownV2)
}

// GenerateAlertsMessage получает алерты и возвращает отформатированное строковое сообщение.
// Эта функция содержит основную логику и не зависит от telebot.
func GenerateAlertsMessage(ctx context.Context, client *monitoring.Client) (string, error) {
	alerts, err := client.GetActiveAlerts(ctx)
	if err != nil {
		return "", fmt.Errorf("ошибка получения активных алертов: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("🔍 *Проверка алертов:*\n\n")

	if len(alerts) > 0 {
		sb.WriteString("🔥 *Активные алерты:*\n")
		for _, alert := range alerts {
			sb.WriteString(fmt.Sprintf("> *%s*\n", escapeMarkdown(alert.Labels["alertname"])))
			if desc, ok := alert.Annotations["description"]; ok {
				sb.WriteString(fmt.Sprintf("  _%s_\n", escapeMarkdown(desc)))
			}
			if severity, ok := alert.Labels["severity"]; ok {
				sb.WriteString(fmt.Sprintf("  Severity: `%s`\n", escapeMarkdown(severity)))
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("✅ *Нет активных алертов*\n")
	}

	return sb.String(), nil
}
