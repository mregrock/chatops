//go:build integration_kube
// +build integration_kube

package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	k8sclient "chatops/internal/kube"
)

const (
	testNamespace      = "test-integration"
	testDeploymentName = "test-deployment"
)

// Вспомогательная функция для создания указателя на int64
func int64Ptr(i int64) *int64 {
	return &i
}

// Вспомогательная функция для создания указателя на int32
func int32Ptr(i int32) *int32 {
	return &i
}

// Функция для получения логов пода
func getPodLogs(client *k8sclient.K8sClient, namespace, podName string) (string, error) {
	req := client.GetClientset().CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Previous:  false,
		TailLines: int64Ptr(10),
	})
	logs, err := req.DoRaw(context.TODO())
	if err != nil {
		return "", fmt.Errorf("ошибка получения логов пода %s: %v", podName, err)
	}
	return string(logs), nil
}

// Функция для получения информации о ReplicaSet'ах
func getReplicaSetsInfo(client *k8sclient.K8sClient, namespace, deploymentName string) (string, error) {
	// Получаем deployment для получения селектора
	dep, err := client.GetClientset().AppsV1().Deployments(namespace).Get(context.TODO(), deploymentName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("ошибка получения deployment: %v", err)
	}

	// Получаем селектор из deployment
	selector, err := metav1.LabelSelectorAsSelector(dep.Spec.Selector)
	if err != nil {
		return "", fmt.Errorf("ошибка создания селектора: %v", err)
	}

	rsList, err := client.GetClientset().AppsV1().ReplicaSets(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return "", err
	}

	var info strings.Builder
	info.WriteString("📊 ReplicaSets:\n")
	for _, rs := range rsList.Items {
		info.WriteString(fmt.Sprintf("  - %s: desired=%d, current=%d, ready=%d, age=%s\n",
			rs.Name,
			*rs.Spec.Replicas,
			rs.Status.Replicas,
			rs.Status.ReadyReplicas,
			time.Since(rs.CreationTimestamp.Time).Round(time.Second)))
	}
	return info.String(), nil
}

func TestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропускаем интеграционный тест в режиме short")
	}

	ctx := context.Background()
	fmt.Println("🚀 Начинаем интеграционный тест...")

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("Ошибка получения домашней директории: %v", err)
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}
	fmt.Printf("📁 Используем kubeconfig: %s\n", kubeconfig)

	client, err := k8sclient.InitClientFromKubeconfig(kubeconfig)
	assert.NoError(t, err)
	fmt.Println("✅ Клиент Kubernetes инициализирован")

	// Проверяем существующие поды в кластере
	fmt.Println("🔍 Проверяем существующие поды в кластере...")
	pods, err := client.GetClientset().CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	assert.NoError(t, err)
	fmt.Printf("📊 Всего подов в кластере: %d\n", len(pods.Items))
	for _, pod := range pods.Items {
		fmt.Printf("  - %s/%s: %s\n", pod.Namespace, pod.Name, pod.Status.Phase)
	}

	// Удаляем namespace test-integration, если он существует
	fmt.Println("🗑️  Удаляем namespace test-integration, если он существует...")
	err = client.GetClientset().CoreV1().Namespaces().Delete(context.TODO(), "test-integration", metav1.DeleteOptions{})
	if err != nil {
		fmt.Printf("⚠️  Ошибка удаления namespace: %v\n", err)
	}

	// Ждем удаления namespace
	fmt.Println("⏳ Ждем удаления namespace...")
	err = wait.PollImmediate(2*time.Second, 1*time.Minute, func() (bool, error) {
		_, err := client.GetClientset().CoreV1().Namespaces().Get(context.TODO(), "test-integration", metav1.GetOptions{})
		if err != nil {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(t, err)
	fmt.Println("✅ Namespace удален")

	// Создаем namespace test-integration
	fmt.Println("📦 Создаем namespace test-integration...")
	_, err = client.GetClientset().CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-integration",
		},
	}, metav1.CreateOptions{})
	assert.NoError(t, err)
	fmt.Println("✅ Namespace создан")

	defer func() {
		fmt.Printf("🧹 Очистка: удаляем namespace %s...\n", testNamespace)
		_ = client.GetClientset().CoreV1().Namespaces().Delete(context.TODO(), testNamespace, metav1.DeleteOptions{})
	}()

	// Создаем deployment test-deployment
	fmt.Println("📦 Создаем deployment test-deployment...")
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-deployment",
			Annotations: map[string]string{
				"deployment.kubernetes.io/revision": "1",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.19",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}
	_, err = client.GetClientset().AppsV1().Deployments("test-integration").Create(context.TODO(), deployment, metav1.CreateOptions{})
	assert.NoError(t, err)
	fmt.Println("✅ Deployment создан")

	// Ждем готовности deployment
	fmt.Println("⏳ Ждем готовности deployment...")
	err = waitForDeploymentReady(client, "test-integration", "test-deployment")
	assert.NoError(t, err)
	fmt.Println("✅ Deployment готов")

	// Проверяем поды
	fmt.Println("🔍 Проверяем поды...")
	pods, err = client.GetClientset().CoreV1().Pods("test-integration").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app=test",
	})
	assert.NoError(t, err)
	assert.Equal(t, 2, len(pods.Items))
	for _, pod := range pods.Items {
		assert.Equal(t, corev1.PodRunning, pod.Status.Phase)
		fmt.Printf("  - %s: %s\n", pod.Name, pod.Status.Phase)
	}

	// Проверяем ReplicaSets
	fmt.Println("🔍 Проверяем ReplicaSets...")
	rsInfo, err := getReplicaSetsInfo(client, "test-integration", "test-deployment")
	assert.NoError(t, err)
	fmt.Println(rsInfo)

	// Масштабируем deployment до 3 реплик
	fmt.Println("📈 Масштабируем deployment до 3 реплик...")
	logCh := make(chan string, 100)
	go func() {
		for msg := range logCh {
			fmt.Println(msg)
		}
	}()
	err = client.ScaleDeploymentWithLogs(ctx, "test-integration", "test-deployment", 3, logCh)
	assert.NoError(t, err)
	close(logCh)

	// Проверяем поды после масштабирования
	fmt.Println("🔍 Проверяем поды после масштабирования...")
	pods, err = client.GetClientset().CoreV1().Pods("test-integration").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app=test",
	})
	assert.NoError(t, err)
	assert.Equal(t, 3, len(pods.Items))
	for _, pod := range pods.Items {
		assert.Equal(t, corev1.PodRunning, pod.Status.Phase)
		fmt.Printf("  - %s: %s\n", pod.Name, pod.Status.Phase)
	}

	// Обновляем образ в deployment
	fmt.Println("🔄 Обновляем образ в deployment...")
	deployment, err = client.GetClientset().AppsV1().Deployments("test-integration").Get(context.TODO(), "test-deployment", metav1.GetOptions{})
	assert.NoError(t, err)
	deployment.Spec.Template.Spec.Containers[0].Image = "nginx:1.20"
	_, err = client.GetClientset().AppsV1().Deployments("test-integration").Update(context.TODO(), deployment, metav1.UpdateOptions{})
	assert.NoError(t, err)

	// Ждем обновления deployment
	fmt.Println("⏳ Ждем обновления deployment...")
	err = waitForDeploymentReady(client, "test-integration", "test-deployment")
	assert.NoError(t, err)
	fmt.Println("✅ Deployment обновлен")

	// Проверяем ReplicaSets после обновления
	fmt.Println("🔍 Проверяем ReplicaSets после обновления...")
	rsInfo, err = getReplicaSetsInfo(client, "test-integration", "test-deployment")
	assert.NoError(t, err)
	fmt.Println(rsInfo)

	// Откат к предыдущей версии
	fmt.Println("⏪ Откат к предыдущей версии...")
	logCh = make(chan string, 100)
	go func() {
		for msg := range logCh {
			fmt.Println(msg)
		}
	}()
	err = client.RollbackDeploymentWithLogs(ctx, "test-integration", "test-deployment", 0, logCh)
	if err != nil {
		t.Fatalf("Ошибка отката deployment: %v", err)
	}
	close(logCh)

	// Проверяем, что откат успешен
	err = waitForDeploymentReady(client, "test-integration", "test-deployment")
	if err != nil {
		t.Fatalf("Ошибка ожидания готовности deployment после отката: %v", err)
	}

	// Проверяем, что версия образа вернулась к предыдущей
	dep, err := client.GetClientset().AppsV1().Deployments("test-integration").Get(ctx, "test-deployment", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Ошибка получения deployment после отката: %v", err)
	}
	if dep.Spec.Template.Spec.Containers[0].Image != "nginx:1.19" {
		t.Errorf("Неверная версия образа после отката: %s", dep.Spec.Template.Spec.Containers[0].Image)
	}

	// Откат к конкретной ревизии
	fmt.Println("⏪ Откат к конкретной ревизии...")
	// Добавляем задержку перед откатом к конкретной ревизии
	time.Sleep(5 * time.Second)
	// Сначала делаем несколько обновлений, чтобы создать историю ревизий
	for i := 0; i < 3; i++ {
		dep, err = client.GetClientset().AppsV1().Deployments("test-integration").Get(ctx, "test-deployment", metav1.GetOptions{})
		if err != nil {
			t.Fatalf("Ошибка получения deployment: %v", err)
		}
		dep.Spec.Template.Spec.Containers[0].Image = fmt.Sprintf("nginx:1.%d", 22+i)
		_, err = client.GetClientset().AppsV1().Deployments("test-integration").Update(ctx, dep, metav1.UpdateOptions{})
		if err != nil {
			t.Fatalf("Ошибка обновления deployment: %v", err)
		}
		time.Sleep(2 * time.Second) // Даем время на применение изменений
	}

	// Откат к ревизии 2 (первое обновление)
	logCh = make(chan string, 100)
	go func() {
		for msg := range logCh {
			fmt.Println(msg)
		}
	}()
	err = client.RollbackDeploymentWithLogs(ctx, "test-integration", "test-deployment", 2, logCh)
	if err != nil {
		t.Fatalf("Ошибка отката к конкретной ревизии: %v", err)
	}
	close(logCh)

	// Проверяем, что откат успешен
	err = waitForDeploymentReady(client, "test-integration", "test-deployment")
	if err != nil {
		t.Fatalf("Ошибка ожидания готовности deployment после отката к конкретной ревизии: %v", err)
	}

	// Проверяем, что версия образа соответствует выбранной ревизии
	dep, err = client.GetClientset().AppsV1().Deployments("test-integration").Get(ctx, "test-deployment", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Ошибка получения deployment после отката к конкретной ревизии: %v", err)
	}
	if dep.Spec.Template.Spec.Containers[0].Image != "nginx:1.20" {
		t.Errorf("Неверная версия образа после отката к конкретной ревизии: %s", dep.Spec.Template.Spec.Containers[0].Image)
	}

	// Перезапускаем deployment
	fmt.Println("🔄 Перезапускаем deployment...")
	logCh = make(chan string, 100)
	go func() {
		for msg := range logCh {
			fmt.Println(msg)
		}
	}()
	err = client.RestartDeploymentWithLogs(ctx, "test-integration", "test-deployment", logCh)
	assert.NoError(t, err)
	close(logCh)

	// Ждем перезапуска deployment
	fmt.Println("⏳ Ждем перезапуска deployment...")
	err = waitForDeploymentReady(client, "test-integration", "test-deployment")
	assert.NoError(t, err)
	fmt.Println("✅ Deployment перезапущен")

	// Проверяем ReplicaSets после перезапуска
	fmt.Println("🔍 Проверяем ReplicaSets после перезапуска...")
	rsInfo, err = getReplicaSetsInfo(client, "test-integration", "test-deployment")
	assert.NoError(t, err)
	fmt.Println(rsInfo)

	// Удаляем namespace test-integration
	fmt.Println("🗑️  Удаляем namespace test-integration...")
	err = client.GetClientset().CoreV1().Namespaces().Delete(context.TODO(), "test-integration", metav1.DeleteOptions{})
	assert.NoError(t, err)
	fmt.Println("✅ Namespace удален")

	fmt.Println("🎉 Интеграционный тест успешно завершен!")
}

func TestIntegrationRollbackAllRevisions(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропускаем интеграционный тест в режиме short")
	}

	ctx := context.Background()
	fmt.Println("🚀 Начинаем интеграционный тест отката на все ревизии...")

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("Ошибка получения домашней директории: %v", err)
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}
	fmt.Printf("📁 Используем kubeconfig: %s\n", kubeconfig)

	client, err := k8sclient.InitClientFromKubeconfig(kubeconfig)
	assert.NoError(t, err)
	fmt.Println("✅ Клиент Kubernetes инициализирован")

	// Удаляем namespace test-integration, если он существует
	fmt.Println("🗑️  Удаляем namespace test-integration, если он существует...")
	err = client.GetClientset().CoreV1().Namespaces().Delete(context.TODO(), "test-integration", metav1.DeleteOptions{})
	if err != nil {
		fmt.Printf("⚠️  Ошибка удаления namespace: %v\n", err)
	}

	// Ждем удаления namespace
	fmt.Println("⏳ Ждем удаления namespace...")
	err = wait.PollImmediate(2*time.Second, 1*time.Minute, func() (bool, error) {
		_, err := client.GetClientset().CoreV1().Namespaces().Get(context.TODO(), "test-integration", metav1.GetOptions{})
		if err != nil {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(t, err)
	fmt.Println("✅ Namespace удален")

	// Создаем namespace test-integration
	fmt.Println("📦 Создаем namespace test-integration...")
	_, err = client.GetClientset().CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-integration",
		},
	}, metav1.CreateOptions{})
	assert.NoError(t, err)
	fmt.Println("✅ Namespace создан")

	defer func() {
		fmt.Printf("🧹 Очистка: удаляем namespace %s...\n", "test-integration")
		_ = client.GetClientset().CoreV1().Namespaces().Delete(context.TODO(), "test-integration", metav1.DeleteOptions{})
	}()

	// Создаем deployment test-deployment
	fmt.Println("📦 Создаем deployment test-deployment...")
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-deployment",
			Annotations: map[string]string{
				"deployment.kubernetes.io/revision": "1",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "test",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "test",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "nginx",
							Image: "nginx:1.19",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}
	_, err = client.GetClientset().AppsV1().Deployments("test-integration").Create(context.TODO(), deployment, metav1.CreateOptions{})
	assert.NoError(t, err)
	fmt.Println("✅ Deployment создан")

	// Ждем готовности deployment
	fmt.Println("⏳ Ждем готовности deployment...")
	err = waitForDeploymentReady(client, "test-integration", "test-deployment")
	assert.NoError(t, err)
	fmt.Println("✅ Deployment готов")

	// Создаем несколько ревизий, обновляя образ
	images := []string{"nginx:1.20", "nginx:1.21", "nginx:1.22"}
	for _, image := range images {
		fmt.Printf("🔄 Обновляем образ на %s...\n", image)
		for i := 0; i < 5; i++ { // Пробуем до 5 раз
			dep, err := client.GetClientset().AppsV1().Deployments("test-integration").Get(ctx, "test-deployment", metav1.GetOptions{})
			assert.NoError(t, err)
			dep.Spec.Template.Spec.Containers[0].Image = image
			_, err = client.GetClientset().AppsV1().Deployments("test-integration").Update(ctx, dep, metav1.UpdateOptions{})
			if err == nil {
				break
			}
			if i < 4 { // Если это не последняя попытка
				fmt.Printf("⚠️  Попытка %d не удалась, пробуем снова...\n", i+1)
				time.Sleep(time.Second) // Ждем секунду перед следующей попыткой
				continue
			}
			assert.NoError(t, err) // Если все попытки не удались, выходим с ошибкой
		}
		err = waitForDeploymentReady(client, "test-integration", "test-deployment")
		assert.NoError(t, err)
		fmt.Printf("✅ Образ обновлен на %s\n", image)
	}

	// Получаем список доступных ревизий
	fmt.Println("📋 Получаем список доступных ревизий...")
	revisions, err := client.ListAvailableRevisions(ctx, "test-integration", "test-deployment")
	assert.NoError(t, err)
	fmt.Printf("📊 Найдено %d ревизий\n", len(revisions))

	// Пытаемся откатиться на каждую ревизию
	for _, rev := range revisions {
		fmt.Printf("⏪ Откат к ревизии %d (RS: %s, Image: %s)...\n", rev.Revision, rev.RSName, rev.Image)
		logCh := make(chan string, 100)
		done := make(chan struct{})
		go func() {
			for msg := range logCh {
				fmt.Println(msg)
			}
			close(done)
		}()

		err := client.RollbackDeploymentWithLogs(ctx, "test-integration", "test-deployment", rev.Revision, logCh)
		assert.NoError(t, err)
		close(logCh)
		<-done

		// Проверяем, что откат успешен
		err = waitForDeploymentReady(client, "test-integration", "test-deployment")
		assert.NoError(t, err)

		// Проверяем, что версия образа соответствует выбранной ревизии
		dep, err := client.GetClientset().AppsV1().Deployments("test-integration").Get(ctx, "test-deployment", metav1.GetOptions{})
		assert.NoError(t, err)
		assert.Equal(t, rev.Image, dep.Spec.Template.Spec.Containers[0].Image)
		fmt.Printf("✅ Откат к ревизии %d успешно завершен\n", rev.Revision)
	}

	fmt.Println("🎉 Интеграционный тест отката на все ревизии успешно завершен!")
}

// Вспомогательная функция для ожидания готовности deployment
func waitForDeploymentReady(client *k8sclient.K8sClient, namespace, name string) error {
	ctx := context.Background()
	return wait.PollUntilContextCancel(ctx, 2*time.Second, true, func(ctx context.Context) (bool, error) {
		dep, err := client.GetClientset().AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		ready := dep.Status.AvailableReplicas == *dep.Spec.Replicas &&
			dep.Status.UpdatedReplicas == *dep.Spec.Replicas &&
			dep.Status.Replicas == *dep.Spec.Replicas &&
			dep.Status.UnavailableReplicas == 0

		if !ready {
			fmt.Printf("⏳ Ожидание готовности: доступно %d/%d реплик (обновлено: %d, всего: %d, недоступно: %d)\n",
				dep.Status.AvailableReplicas, *dep.Spec.Replicas,
				dep.Status.UpdatedReplicas, dep.Status.Replicas,
				dep.Status.UnavailableReplicas)
		}

		return ready, nil
	})
}

func TestIntegrationGetPodLogs(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропускаем интеграционный тест в режиме short")
	}

	ctx := context.Background()
	fmt.Println("🚀 Начинаем интеграционный тест получения логов...")

	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("Ошибка получения домашней директории: %v", err)
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}
	fmt.Printf("📁 Используем kubeconfig: %s\n", kubeconfig)

	client, err := k8sclient.InitClientFromKubeconfig(kubeconfig)
	assert.NoError(t, err)
	fmt.Println("✅ Клиент Kubernetes инициализирован")

	// Удаляем namespace test-integration, если он существует
	fmt.Println("🗑️  Удаляем namespace test-integration, если он существует...")
	err = client.GetClientset().CoreV1().Namespaces().Delete(context.TODO(), "test-integration", metav1.DeleteOptions{})
	if err != nil {
		fmt.Printf("⚠️  Ошибка удаления namespace: %v\n", err)
	}

	// Ждем удаления namespace
	fmt.Println("⏳ Ждем удаления namespace...")
	err = wait.PollImmediate(2*time.Second, 1*time.Minute, func() (bool, error) {
		_, err := client.GetClientset().CoreV1().Namespaces().Get(context.TODO(), "test-integration", metav1.GetOptions{})
		if err != nil {
			return true, nil
		}
		return false, nil
	})
	assert.NoError(t, err)
	fmt.Println("✅ Namespace удален")

	// Создаем namespace test-integration
	fmt.Println("📦 Создаем namespace test-integration...")
	_, err = client.GetClientset().CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-integration",
		},
	}, metav1.CreateOptions{})
	assert.NoError(t, err)
	fmt.Println("✅ Namespace создан")

	defer func() {
		fmt.Printf("🧹 Очистка: удаляем namespace %s...\n", "test-integration")
		_ = client.GetClientset().CoreV1().Namespaces().Delete(context.TODO(), "test-integration", metav1.DeleteOptions{})
	}()

	// Создаем под с nginx
	fmt.Println("📦 Создаем под с nginx...")
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-nginx-pod",
			Namespace: "test-integration",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx:latest",
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 80,
						},
					},
				},
			},
		},
	}

	_, err = client.GetClientset().CoreV1().Pods("test-integration").Create(context.TODO(), pod, metav1.CreateOptions{})
	assert.NoError(t, err)
	fmt.Println("✅ Под создан")

	// Ждем, пока под будет готов
	fmt.Println("⏳ Ждем готовности пода...")
	err = wait.PollImmediate(2*time.Second, 2*time.Minute, func() (bool, error) {
		pod, err := client.GetClientset().CoreV1().Pods("test-integration").Get(context.TODO(), "test-nginx-pod", metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		return pod.Status.Phase == corev1.PodRunning, nil
	})
	assert.NoError(t, err)
	fmt.Println("✅ Под готов")

	// Получаем логи пода
	fmt.Println("📋 Получаем логи пода...")
	logs, err := client.GetPodLogs(ctx, "test-integration", "test-nginx-pod", &k8sclient.PodLogsOptions{
		TailLines:    10,
		SinceSeconds: 3600,
		Timestamps:   true,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, logs)
	fmt.Printf("📝 Получены логи пода:\n%s\n", logs)

	// Проверяем, что логи содержат ожидаемые строки
	assert.Contains(t, logs, "nginx")
	assert.Contains(t, logs, "worker process")

	fmt.Println("🎉 Интеграционный тест получения логов успешно завершен!")
}
