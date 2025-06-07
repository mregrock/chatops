package app_test

import (
	"bytes"
	"chatops/internal/app"
	"chatops/internal/monitoring"
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"testing"

	"chatops/internal/db/models"
)

type mockMonitoringClient struct {
	Alerts []monitoring.Alert
	Error  error
}

func (m *mockMonitoringClient) GetActiveAlerts(ctx context.Context) ([]monitoring.Alert, error) {
	return m.Alerts, m.Error
}

type mockDutyFinder struct {
	Users []models.User
	Error error
}

func (m *mockDutyFinder) GetDutyUsersByLabel(label string) ([]models.User, error) {
	if m.Error == nil && len(m.Users) > 0 && label != "" {
		return m.Users, nil
	}
	return nil, m.Error
}

func TestAlerter_CheckAndNotify(t *testing.T) {
	originalLogger := log.Default().Writer()
	defer log.SetOutput(originalLogger)

	testCases := []struct {
		name          string
		mockMonClient app.MonitoringClient
		mockDBClient  app.DutyFinder
		expectedLog   string
		shouldFindLog bool
		expectError   bool
	}{
		{
			name:          "No active alerts",
			mockMonClient: &mockMonitoringClient{Alerts: []monitoring.Alert{}},
			mockDBClient:  &mockDutyFinder{},
			expectedLog:   "No active alerts found",
			shouldFindLog: true,
			expectError:   false,
		},
		{
			name: "One alert, one duty user found",
			mockMonClient: &mockMonitoringClient{
				Alerts: []monitoring.Alert{{Labels: map[string]string{"job": "test"}}},
			},
			mockDBClient: &mockDutyFinder{
				Users: []models.User{{Login: "test-user"}},
			},
			expectedLog:   "УВЕДОМЛЕНИЕ ДЛЯ: @test-user",
			shouldFindLog: true,
			expectError:   false,
		},
		{
			name: "One alert, but no duty user found",
			mockMonClient: &mockMonitoringClient{
				Alerts: []monitoring.Alert{{Labels: map[string]string{"job": "unknown"}}},
			},
			mockDBClient:  &mockDutyFinder{Users: []models.User{}},
			expectedLog:   "No duty users found for alert",
			shouldFindLog: true,
			expectError:   false,
		},
		{
			name: "One alert, but DB returns an error",
			mockMonClient: &mockMonitoringClient{
				Alerts: []monitoring.Alert{{Labels: map[string]string{"job": "any"}}},
			},
			mockDBClient:  &mockDutyFinder{Error: errors.New("connection failed")},
			expectedLog:   "Error searching duty users",
			shouldFindLog: true,
			expectError:   false,
		},
		{
			name:          "Error from monitoring client",
			mockMonClient: &mockMonitoringClient{Error: errors.New("prom-error")},
			mockDBClient:  &mockDutyFinder{},
			expectedLog:   "",
			shouldFindLog: false,
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var logBuffer bytes.Buffer
			log.SetOutput(&logBuffer)

			alerter := app.NewAlerter(tc.mockMonClient, tc.mockDBClient)
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
