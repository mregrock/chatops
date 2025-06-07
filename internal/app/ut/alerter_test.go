package app_test

import (
	"bytes"
	"context"
	"errors"
	"hackaton/internal/app"
	"hackaton/internal/monitoring"
	"log"
	"os"
	"strings"
	"testing"
)

type MockMonitoringClient struct {
	Alerts []monitoring.Alert
	Error  error
}

func (m *MockMonitoringClient) GetActiveAlerts(ctx context.Context) ([]monitoring.Alert, error) {
	return m.Alerts, m.Error
}

func TestAlerter_CheckAndNotify(t *testing.T) {
	originalLogger := log.Default().Writer()
	defer log.SetOutput(originalLogger)

	testCases := []struct {
		name          string
		mockClient    *MockMonitoringClient
		expectedLog   string
		shouldFindLog bool
		expectError   bool
	}{
		{
			name: "No active alerts",
			mockClient: &MockMonitoringClient{
				Alerts: []monitoring.Alert{},
				Error:  nil,
			},
			expectedLog:   "No active alerts found",
			shouldFindLog: true,
			expectError:   false,
		},
		{
			name: "One active alert",
			mockClient: &MockMonitoringClient{
				Alerts: []monitoring.Alert{
					{
						Labels: map[string]string{
							"alertname": "TestAppHighErrorRate",
							"job":       "test-app-go-svc",
						},
						Annotations: map[string]string{
							"summary": "High error rate!",
						},
					},
				},
				Error: nil,
			},
			expectedLog:   "УВЕДОМЛЕНИЕ ДЛЯ: Дежурный команды Альфа (Вася)",
			shouldFindLog: true,
			expectError:   false,
		},
		{
			name: "Alert without job label",
			mockClient: &MockMonitoringClient{
				Alerts: []monitoring.Alert{
					{
						Labels: map[string]string{"alertname": "some-other-alert"},
					},
				},
			},
			expectedLog:   "Skipping alert, 'job' label not found",
			shouldFindLog: true,
			expectError:   false,
		},
		{
			name: "Error from monitoring client",
			mockClient: &MockMonitoringClient{
				Error: errors.New("connection refused"),
			},
			expectedLog:   "",
			shouldFindLog: false,
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var logBuffer bytes.Buffer
			log.SetOutput(&logBuffer)

			alerter := app.NewAlerter(tc.mockClient)
			err := alerter.CheckAndNotify(context.Background())

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected an error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, but got: %v", err)
				}
			}

			logOutput := logBuffer.String()
			if tc.shouldFindLog && !strings.Contains(logOutput, tc.expectedLog) {
				t.Errorf("Expected log to contain '%s', but it was: \n%s", tc.expectedLog, logOutput)
			}
		})
	}
}

func init() {
	log.SetOutput(os.Stderr)
}
