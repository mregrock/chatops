package app

import (
	"context"
	"fmt"
	"hackaton/internal/monitoring"
	"log"
	"strings"

	"db/models"
)

// MonitoringClient определяет интерфейс, который нужен Alerter'у от клиента мониторинга.
type MonitoringClient interface {
	GetActiveAlerts(ctx context.Context) ([]monitoring.Alert, error)
}

// DutyFinder определяет интерфейс для получения дежурных из источника данных (например, БД).
// Это позволяет нам легко подменять реальную БД на мок в юнит-тестах.
type DutyFinder interface {
	GetDutyUsersByLabel(label string) ([]models.User, error)
}

// Alerter проверяет алерты и уведомляет дежурных.
type Alerter struct {
	monClient MonitoringClient
	db        DutyFinder
}

// NewAlerter создает новый экземпляр Alerter.
func NewAlerter(monClient MonitoringClient, db DutyFinder) *Alerter {
	return &Alerter{
		monClient: monClient,
		db:        db,
	}
}

// CheckAndNotify получает алерты, находит ответственных в БД и "отправляет" уведомление.
func (a *Alerter) CheckAndNotify(ctx context.Context) error {
	log.Println("Checking for active alerts...")

	alerts, err := a.monClient.GetActiveAlerts(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active alerts: %w", err)
	}

	if len(alerts) == 0 {
		log.Println("No active alerts found.")
		return nil
	}

	log.Printf("Found %d active alerts. Processing...\n", len(alerts))

	for _, alert := range alerts {
		var dutyUsers []models.User
		var err error

		// Пытаемся найти дежурных по меткам из алерта.
		for key, value := range alert.Labels {
			labelToSearch := fmt.Sprintf("%s=%s", key, value)
			// ИСПОЛЬЗУЕМ НАШ ИНТЕРФЕЙС!
			dutyUsers, err = a.db.GetDutyUsersByLabel(labelToSearch)
			if err != nil {
				log.Printf("Error searching duty users for label %s: %v", labelToSearch, err)
				continue
			}
			if len(dutyUsers) > 0 {
				log.Printf("Found %d duty users for label '%s'", len(dutyUsers), labelToSearch)
				break
			}
		}

		if len(dutyUsers) == 0 {
			log.Printf("No duty users found for alert with labels: %v. Skipping.", alert.Labels)
			continue
		}

		// Формируем и "отправляем" уведомление каждому найденному дежурному.
		for _, user := range dutyUsers {
			notification := formatNotification(user.Login, alert)
			log.Println(notification)
		}
	}
	return nil
}
func formatNotification(dutyPersonUsername string, alert monitoring.Alert) string {
	var details []string
	for key, value := range alert.Labels {
		details = append(details, fmt.Sprintf("- %s: %s", key, value))
	}
	labelsFormatted := strings.Join(details, "\n")

	summary := alert.Annotations["summary"]
	description := alert.Annotations["description"]

	return fmt.Sprintf(
		"УВЕДОМЛЕНИЕ ДЛЯ: @%s\n"+
			"==================================\n"+
			"🚨 Сработал алерт: %s\n\n"+
			"📋 Описание: %s\n"+
			"📝 Дополнительно: %s\n\n"+
			"🏷 Метки:\n"+
			"%s\n"+
			"==================================",
		dutyPersonUsername,
		alert.Labels["alertname"],
		summary,
		description,
		labelsFormatted,
	)
}
