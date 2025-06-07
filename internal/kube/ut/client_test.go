package k8sclient

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"chatops/internal/kube"
)

// Переменная clientset для тестов
var testClient *kube.K8sClient

func TestInitClientFromKubeconfig(t *testing.T) {
	tests := []struct {
		name        string
		kubeconfig  string
		expectError bool
	}{
		{
			name:        "Несуществующий kubeconfig",
			kubeconfig:  "/nonexistent/path",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := kube.InitClientFromKubeconfig(tt.kubeconfig)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetPodStatus(t *testing.T) {
	// Создаем тестовый под
	testPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	// Создаем фейковый клиент
	fakeClient := fake.NewSimpleClientset(testPod)
	testClient = kube.NewTestClient(fakeClient)

	tests := []struct {
		name        string
		namespace   string
		podName     string
		expected    string
		expectError bool
	}{
		{
			name:        "Успешное получение статуса",
			namespace:   "test-namespace",
			podName:     "test-pod",
			expected:    "Running",
			expectError: false,
		},
		{
			name:        "Под не найден",
			namespace:   "test-namespace",
			podName:     "nonexistent-pod",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := testClient.GetPodStatus(tt.namespace, tt.podName)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, status)
			}
		})
	}
}

func TestScaleDeploymentWithLogs(t *testing.T) {
	// Создаем тестовый deployment
	testDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "test-namespace",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
		},
	}

	// Создаем фейковый клиент
	fakeClient := fake.NewSimpleClientset(testDeployment)
	testClient = kube.NewTestClient(fakeClient)

	logCh := make(chan string, 10)

	tests := []struct {
		name         string
		namespace    string
		deployName   string
		replicas     int32
		expectError  bool
		expectedLogs []string
	}{
		{
			name:        "Успешное масштабирование",
			namespace:   "test-namespace",
			deployName:  "test-deployment",
			replicas:    3,
			expectError: false,
			expectedLogs: []string{
				"🔧 Масштабируем Deployment test-deployment в namespace test-namespace до 3 реплик...",
				"🚀 Масштабирование запущено...",
				"✅ Масштабирование завершено успешно.",
			},
		},
		{
			name:        "Deployment не найден",
			namespace:   "test-namespace",
			deployName:  "nonexistent-deployment",
			replicas:    3,
			expectError: true,
			expectedLogs: []string{
				"Ошибка получения Deployment: deployments.apps \"nonexistent-deployment\" not found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очищаем канал логов
			for len(logCh) > 0 {
				<-logCh
			}

			// Для успешного теста обновляем статус deployment
			if !tt.expectError {
				go func() {
					time.Sleep(100 * time.Millisecond)
					dep, _ := fakeClient.AppsV1().Deployments(tt.namespace).Get(context.TODO(), tt.deployName, metav1.GetOptions{})
					dep.Status.AvailableReplicas = tt.replicas
					dep.Status.UpdatedReplicas = tt.replicas
					dep.Status.Replicas = tt.replicas
					dep.Status.ObservedGeneration = dep.Generation
					dep.Status.UnavailableReplicas = 0
					fakeClient.AppsV1().Deployments(tt.namespace).UpdateStatus(context.TODO(), dep, metav1.UpdateOptions{})
				}()
			}

			ctx := context.Background()
			err := testClient.ScaleDeploymentWithLogs(ctx, tt.namespace, tt.deployName, tt.replicas, logCh)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Собираем все логи из канала
			var logs []string
			collectTimeout := time.After(time.Second)
		collectLoop:
			for {
				select {
				case log := <-logCh:
					logs = append(logs, log)
				case <-collectTimeout:
					break collectLoop
				}
			}

			// Проверяем, что каждый ожидаемый лог содержится хотя бы в одном из сообщений
			for _, expectedLog := range tt.expectedLogs {
				found := false
				for _, log := range logs {
					if strings.Contains(log, expectedLog) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Log not found: %s", expectedLog)
				}
			}
		})
	}
}

func TestRestartDeploymentWithLogs(t *testing.T) {
	// Создаем тестовый deployment
	testDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "test-namespace",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
		},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			UpdatedReplicas:   1,
			Replicas:          1,
		},
	}

	// Создаем фейковый клиент
	fakeClient := fake.NewSimpleClientset(testDeployment)
	testClient = kube.NewTestClient(fakeClient)

	logCh := make(chan string, 10)

	tests := []struct {
		name         string
		namespace    string
		deployName   string
		expectError  bool
		expectedLogs []string
	}{
		{
			name:        "Успешный рестарт",
			namespace:   "test-namespace",
			deployName:  "test-deployment",
			expectError: false,
			expectedLogs: []string{
				"🚀 Rollout restart запущен...",
				"✅ Rollout завершён успешно.",
			},
		},
		{
			name:        "Deployment не найден",
			namespace:   "test-namespace",
			deployName:  "nonexistent-deployment",
			expectError: true,
			expectedLogs: []string{
				"Ошибка получения Deployment: deployments.apps \"nonexistent-deployment\" not found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очищаем канал логов
			for len(logCh) > 0 {
				<-logCh
			}

			// Для успешного теста обновляем статус deployment
			if !tt.expectError {
				go func() {
					time.Sleep(100 * time.Millisecond)
					dep, _ := fakeClient.AppsV1().Deployments(tt.namespace).Get(context.TODO(), tt.deployName, metav1.GetOptions{})
					dep.Status.AvailableReplicas = 1
					dep.Status.UpdatedReplicas = 1
					dep.Status.Replicas = 1
					dep.Status.ObservedGeneration = dep.Generation
					dep.Status.UnavailableReplicas = 0
					fakeClient.AppsV1().Deployments(tt.namespace).UpdateStatus(context.TODO(), dep, metav1.UpdateOptions{})
				}()
			}

			ctx := context.Background()
			err := testClient.RestartDeploymentWithLogs(ctx, tt.namespace, tt.deployName, logCh)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Собираем все логи из канала
			var logs []string
			collectTimeout := time.After(time.Second)
		collectLoop:
			for {
				select {
				case log := <-logCh:
					logs = append(logs, log)
				case <-collectTimeout:
					break collectLoop
				}
			}

			// Проверяем, что каждый ожидаемый лог содержится хотя бы в одном из сообщений
			for _, expectedLog := range tt.expectedLogs {
				found := false
				for _, log := range logs {
					if strings.Contains(log, expectedLog) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Log not found: %s", expectedLog)
				}
			}
		})
	}
}

func TestInitInCluster(t *testing.T) {
	tests := []struct {
		name        string
		expectError bool
	}{
		{
			name:        "Инициализация внутри кластера",
			expectError: true, // Ожидаем ошибку, так как тест запускается не в кластере
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := kube.InitInCluster()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Вспомогательная функция для создания указателя на int32
func int32Ptr(i int32) *int32 {
	return &i
}
