package app

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"chatops/internal/monitoring"
)

type AlertPoller struct {
	monitoringClient *monitoring.Client
	interval         time.Duration
	ctx              context.Context
	cancelFunc       context.CancelFunc
	wg               sync.WaitGroup
}

func NewAlertPoller(client *monitoring.Client, interval time.Duration) *AlertPoller {
	ctx, cancel := context.WithCancel(context.Background())
	return &AlertPoller{
		monitoringClient: client,
		interval:         interval,
		ctx:              ctx,
		cancelFunc:       cancel,
	}
}

func (p *AlertPoller) Start() {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()

		// Первая проверка при старте
		if err := p.checkAlerts(); err != nil {
			log.Printf("Error checking alerts: %v", err)
		}

		for {
			select {
			case <-p.ctx.Done():
				log.Println("Alert poller stopping...")
				return
			case <-ticker.C:
				if err := p.checkAlerts(); err != nil {
					log.Printf("Error checking alerts: %v", err)
				}
			}
		}
	}()
}

func (p *AlertPoller) Stop() {
	p.cancelFunc()
	p.wg.Wait()
}

func (p *AlertPoller) checkAlerts() error {
	alerts, err := p.monitoringClient.GetActiveAlerts(p.ctx)
	if err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("🔍 *Проверка алертов:*\n\n")

	if len(alerts) > 0 {
		sb.WriteString("🔥 *Активные алерты:*\n")
		for _, alert := range alerts {
			sb.WriteString(fmt.Sprintf("> *%s*\n", alert.Labels["alertname"]))
			if desc, ok := alert.Annotations["description"]; ok {
				sb.WriteString(fmt.Sprintf("  _%s_\n", desc))
			}
			if severity, ok := alert.Labels["severity"]; ok {
				sb.WriteString(fmt.Sprintf("  Severity: `%s`\n", severity))
			}
			sb.WriteString("\n")
		}
	} else {
		sb.WriteString("✅ *Нет активных алертов*\n")
	}

	log.Println(sb.String())
	return nil
}
