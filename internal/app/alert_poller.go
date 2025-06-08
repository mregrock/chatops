package app

import (
	"context"
	"log"
	"sync"
	"time"
)

type AlertPoller struct {
	alerter    *Alerter
	interval   time.Duration
	ctx        context.Context
	cancelFunc context.CancelFunc
	wg         sync.WaitGroup
}

func NewAlertPoller(alerter *Alerter, interval time.Duration) *AlertPoller {
	ctx, cancel := context.WithCancel(context.Background())
	return &AlertPoller{
		alerter:    alerter,
		interval:   interval,
		ctx:        ctx,
		cancelFunc: cancel,
	}
}

func (p *AlertPoller) Start() {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()

		if err := p.alerter.CheckAndNotify(p.ctx); err != nil {
			log.Printf("Error checking alerts: %v", err)
		}

		for {
			select {
			case <-p.ctx.Done():
				log.Println("Alert poller stopping...")
				return
			case <-ticker.C:
				if err := p.alerter.CheckAndNotify(p.ctx); err != nil {
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

