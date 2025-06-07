package app

import (
	"context"
	"fmt"
	"hackaton/internal/monitoring"
	"log"
)

type MonitoringClient interface {
	GetActiveAlerts(ctx context.Context) ([]monitoring.Alert, error)
}

type Alerter struct {
	monClient MonitoringClient
}

func NewAlerter(monClient MonitoringClient) *Alerter {
	return &Alerter{
		monClient: monClient,
	}
}

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
		serviceName, ok := alert.Labels["job"]
		if !ok {
			log.Printf("Skipping alert, 'job' label not found: %v", alert.Labels)
			continue
		}

		dutyPerson, err := getOnCallDuty(serviceName)
		if err != nil {
			log.Printf("Could not get duty person for service %s: %v. Skipping.", serviceName, err)
			continue
		}

		notification := formatNotification(dutyPerson, serviceName, alert)
		log.Println(notification)
	}
	return nil
}

func getOnCallDuty(serviceName string) (string, error) {
	switch serviceName {
	case "test-app-go-svc":
		return "Дежурный команды Альфа (Вася)", nil
	case "prometheus":
		return "Дежурный по инфраструктуре (Петя)", nil
	default:
		return "Главный дежурный (Иван)", nil
	}
}

func formatNotification(dutyPerson, serviceName string, alert monitoring.Alert) string {
	summary := alert.Annotations["summary"]
	if summary == "" {
		summary = "No summary provided."
	}
	return fmt.Sprintf(
		"УВЕДОМЛЕНИЕ ДЛЯ: %s\n"+
			"==================================\n"+
			"🚨 Сработал алерт!\n"+
			"Сервис: %s\n"+
			"Название алерта: %s\n"+
			"Описание: %s\n"+
			"==================================",
		dutyPerson,
		serviceName,
		alert.Labels["alertname"],
		summary,
	)
}
