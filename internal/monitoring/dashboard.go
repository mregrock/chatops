package monitoring

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

type ServiceStatusDashboard struct {
	ServiceName string
	Alerts      []Alert
	Pods        []PodStatus
}

type PodStatus struct {
	PodName          string
	Phase            string  // Фаза жизненного цикла пода (e.g., Running, Pending)
	Ready            bool    // Готов ли под к приему трафика
	CPUUsageCores    float64 // Текущее потребление CPU в ядрах
	CPULimitCores    float64 // Лимит CPU в ядрах
	MemoryUsageBytes float64 // Текущее потребление памяти в байтах
	MemoryLimitBytes float64 // Лимит памяти в байтах
	Restarts         int64   // Количество перезапусков
	OOMKilled        bool    // Был ли под убит по OOM
}

func (c *Client) GetStatusDashboard(ctx context.Context, namespace, jobName string) (*ServiceStatusDashboard, error) {
	dashboard := &ServiceStatusDashboard{
		ServiceName: jobName,
		Pods:        []PodStatus{},
	}

	podNames, err := c.getPodNamesForJob(ctx, jobName)
	if err != nil {
		return nil, fmt.Errorf("could not get pod names for job %s: %w", jobName, err)
	}
	if len(podNames) == 0 {
		return dashboard, nil
	}

	allAlerts, err := c.GetActiveAlerts(ctx)
	if err != nil {
		fmt.Printf("Warning: could not get active alerts: %v\n", err)
	} else {
		for _, alert := range allAlerts {
			if alert.Labels["job"] == jobName && (namespace == "" || alert.Labels["namespace"] == namespace) {
				dashboard.Alerts = append(dashboard.Alerts, alert)
			}
		}
	}

	podMetrics := make(map[string]*PodStatus)
	for _, name := range podNames {
		podMetrics[name] = &PodStatus{PodName: name}
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	queryAndParse := func(query string, parser func(*PodStatus, float64)) {
		defer wg.Done()
		resp, err := c.Query(ctx, query)
		if err != nil {
			fmt.Printf("Error querying prometheus for job %s: %v\n", jobName, err)
			return
		}
		if resp.Data.ResultType != "vector" {
			return
		}

		mu.Lock()
		defer mu.Unlock()
		for _, result := range resp.Data.Result {
			podName, ok := result.Metric["pod"]
			if !ok {
				continue
			}
			if podStatus, exists := podMetrics[podName]; exists {
				value, err := strconv.ParseFloat(result.Value[1].(string), 64)
				if err != nil {
					continue
				}
				parser(podStatus, value)
			}
		}
	}

	queryAndParseLabel := func(query, labelName string, parser func(*PodStatus, string)) {
		defer wg.Done()
		resp, err := c.Query(ctx, query)
		if err != nil {
			fmt.Printf("Error querying prometheus for job %s: %v\n", jobName, err)
			return
		}
		if resp.Data.ResultType != "vector" {
			return
		}

		mu.Lock()
		defer mu.Unlock()
		for _, result := range resp.Data.Result {
			podName, ok := result.Metric["pod"]
			if !ok {
				continue
			}
			if podStatus, exists := podMetrics[podName]; exists {
				if labelValue, ok := result.Metric[labelName]; ok {
					parser(podStatus, labelValue)
				}
			}
		}
	}

	podsRegex := strings.Join(podNames, "|")

	queries := map[string]func(*PodStatus, float64){
		fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{pod=~"%s", container!="", image!~".*pause.*"}[5m])) by (pod)`, podsRegex): func(ps *PodStatus, v float64) {
			ps.CPUUsageCores = v
		},
		fmt.Sprintf(`sum(kube_pod_container_resource_limits{pod=~"%s", resource="cpu"}) by (pod)`, podsRegex): func(ps *PodStatus, v float64) {
			ps.CPULimitCores = v
		},
		fmt.Sprintf(`sum(container_memory_working_set_bytes{pod=~"%s", container!="", image!~".*pause.*"}) by (pod)`, podsRegex): func(ps *PodStatus, v float64) {
			ps.MemoryUsageBytes = v
		},
		fmt.Sprintf(`sum(kube_pod_container_resource_limits{pod=~"%s", resource="memory"}) by (pod)`, podsRegex): func(ps *PodStatus, v float64) {
			ps.MemoryLimitBytes = v
		},
		fmt.Sprintf(`sum(kube_pod_container_status_restarts_total{pod=~"%s"}) by (pod)`, podsRegex): func(ps *PodStatus, v float64) {
			ps.Restarts = int64(v)
		},
		fmt.Sprintf(`kube_pod_status_ready{condition="true", pod=~"%s"}`, podsRegex): func(ps *PodStatus, v float64) {
			if v == 1 {
				ps.Ready = true
			}
		},
	}

	for query, parser := range queries {
		wg.Add(1)
		go queryAndParse(query, parser)
	}

	wg.Add(1)
	go queryAndParseLabel(
		fmt.Sprintf(`kube_pod_status_phase{pod=~"%s"} > 0`, podsRegex),
		"phase",
		func(ps *PodStatus, phase string) {
			ps.Phase = phase
		},
	)

	wg.Add(1)
	go func() {
		defer wg.Done()
		oomQuery := fmt.Sprintf(`kube_pod_container_status_last_terminated_reason{pod=~"%s", reason="OOMKilled"}`, podsRegex)
		resp, err := c.Query(ctx, oomQuery)
		if err != nil {
			fmt.Printf("Error querying OOMKilled for job %s: %v\n", jobName, err)
			return
		}
		if resp.Data.ResultType != "vector" {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		for _, result := range resp.Data.Result {
			podName, ok := result.Metric["pod"]
			if !ok {
				continue
			}
			if podStatus, exists := podMetrics[podName]; exists {
				podStatus.OOMKilled = true
			}
		}
	}()

	wg.Wait()

	for _, podStatus := range podMetrics {
		dashboard.Pods = append(dashboard.Pods, *podStatus)
	}

	return dashboard, nil
}

func (c *Client) getPodNamesForJob(ctx context.Context, jobName string) ([]string, error) {
	query := fmt.Sprintf(`up{job="%s"}`, jobName)
	resp, err := c.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	if resp.Data.ResultType != "vector" {
		return nil, fmt.Errorf("unexpected result type for pod name query: %s", resp.Data.ResultType)
	}

	podNames := make(map[string]struct{})
	for _, result := range resp.Data.Result {
		if podName, ok := result.Metric["pod"]; ok {
			podNames[podName] = struct{}{}
		}
	}

	namesList := make([]string, 0, len(podNames))
	for name := range podNames {
		namesList = append(namesList, name)
	}

	return namesList, nil
}
