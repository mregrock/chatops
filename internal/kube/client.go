package kube

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// RevisionInfo содержит информацию о доступной ревизии для отката
type RevisionInfo struct {
	Revision int64
	RSName   string
	Image    string
}

// PodLogsOptions содержит опции для получения логов пода
type PodLogsOptions struct {
	// Previous получает логи предыдущего контейнера
	Previous bool
	// TailLines ограничивает количество последних строк логов
	TailLines int64
	// SinceSeconds ограничивает логи временным интервалом в секундах
	SinceSeconds int64
	// Timestamps добавляет временные метки к логам
	Timestamps bool
}

type K8sClientInterface interface {
	GetPodStatus(ctx context.Context, namespace, podName string) (string, error)
	ScaleDeploymentWithLogs(ctx context.Context, namespace, name string, replicas int32, logCh chan<- string) error
	RollbackDeploymentWithLogs(ctx context.Context, namespace, name string, revision int64, logCh chan<- string) error
	RestartDeploymentWithLogs(ctx context.Context, namespace, name string, logCh chan<- string) error
	ListAvailableRevisions(ctx context.Context, namespace, deploymentName string) ([]RevisionInfo, error)
	GetClientset() kubernetes.Interface
	GetPodLogs(ctx context.Context, namespace, podName string, opts *PodLogsOptions) (string, error)
}

type K8sClient struct {
	clientset kubernetes.Interface
}

// NewTestClient создает новый тестовый клиент
func NewTestClient(clientset kubernetes.Interface) *K8sClient {
	return &K8sClient{clientset: clientset}
}

// InitClientFromKubeconfig инициализирует client-go из kubeconfig
func InitClientFromKubeconfig(path string) (*K8sClient, error) {
	config, err := clientcmd.BuildConfigFromFlags("", path)
	if err != nil {
		return nil, err
	}

	return initClient(config)
}

// InitInCluster инициализирует client-go внутри кластера
func InitInCluster() (*K8sClient, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	return initClient(config)
}

func initClient(config *rest.Config) (*K8sClient, error) {
	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &K8sClient{clientset: cs}, nil
}

func (c *K8sClient) GetPodStatus(ctx context.Context, namespace, podName string) (string, error) {
	pod, err := c.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return string(pod.Status.Phase), nil
}

/*func ScaleDeployment(namespace, name string, replicas int32) error {
	dep, err := clientset.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}

	dep.Spec.Replicas = &replicas
	_, err = clientset.AppsV1().Deployments(namespace).Update(context.TODO(), dep, metav1.UpdateOptions{})
	return err
}*/

func (c *K8sClient) ScaleDeploymentWithLogs(ctx context.Context, namespace, name string, replicas int32, logCh chan<- string) error {
	if c.clientset == nil {
		return fmt.Errorf("client not initialized")
	}

	log := func(msg string) {
		if logCh != nil {
			logCh <- msg
		}
	}

	// Получаем Deployment
	dep, err := c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log(fmt.Sprintf("Ошибка получения Deployment: %v", err))
		return err
	}

	log(fmt.Sprintf("🔧 Масштабируем Deployment %s в namespace %s до %d реплик...", name, namespace, replicas))

	dep.Spec.Replicas = &replicas

	_, err = c.clientset.AppsV1().Deployments(namespace).Update(ctx, dep, metav1.UpdateOptions{})
	if err != nil {
		log(fmt.Sprintf("Ошибка обновления Deployment: %v", err))
		return err
	}

	log("🚀 Масштабирование запущено...")

	// Ждём, пока Deployment достигнет нужного количества доступных реплик
	return wait.PollUntilContextCancel(ctx, 2*time.Second, true, func(ctx context.Context) (bool, error) {
		updated, err := c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			log(fmt.Sprintf("Ошибка чтения Deployment: %v", err))
			return false, err
		}

		log(fmt.Sprintf(
			"🌀 Статус: доступно %d/%d реплик",
			updated.Status.AvailableReplicas,
			replicas,
		))

		if updated.Status.AvailableReplicas == replicas {
			log("✅ Масштабирование завершено успешно.")
			return true, nil
		}

		return false, nil
	})
}

/*func RollbackDeployment(namespace, name string) error {
	// Kubernetes не поддерживает роллбек прямо через API в новых версиях.
	// Реализуется через сохранение предыдущего ReplicaSet и его масштабирование вручную.
	// Либо используйте `kubectl rollout undo` через exec, либо кастомная логика.
	return nil // Можно реализовать если надо
}*/

func (c *K8sClient) RollbackDeploymentWithLogs(ctx context.Context, namespace, name string, revision int64, logCh chan<- string) error {
	log := func(msg string) {
		if logCh != nil {
			logCh <- msg
		}
	}

	log("[rollback] Получаем deployment...")
	dep, err := c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log(fmt.Sprintf("[rollback] Ошибка получения deployment: %v", err))
		return fmt.Errorf("ошибка получения deployment: %v", err)
	}

	log("[rollback] Получаем селектор из deployment...")
	selector, err := metav1.LabelSelectorAsSelector(dep.Spec.Selector)
	if err != nil {
		log(fmt.Sprintf("[rollback] Ошибка создания селектора: %v", err))
		return fmt.Errorf("ошибка создания селектора: %v", err)
	}

	log(fmt.Sprintf("[rollback] Селектор: %s", selector.String()))
	log("[rollback] Получаем список ReplicaSets...")
	rsList, err := c.clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		log(fmt.Sprintf("[rollback] Ошибка получения списка ReplicaSets: %v", err))
		return fmt.Errorf("ошибка получения списка ReplicaSets: %v", err)
	}

	log(fmt.Sprintf("[rollback] Найдено ReplicaSets: %d", len(rsList.Items)))
	for i, rs := range rsList.Items {
		log(fmt.Sprintf("[rollback] RS[%d]: %s, ревизия: %s, replicas: %d, ready: %d, created: %s", i, rs.Name, rs.Annotations["deployment.kubernetes.io/revision"], rs.Status.Replicas, rs.Status.ReadyReplicas, rs.CreationTimestamp.Time.Format(time.RFC3339)))
	}

	sort.Slice(rsList.Items, func(i, j int) bool {
		return rsList.Items[i].CreationTimestamp.After(rsList.Items[j].CreationTimestamp.Time)
	})

	if len(rsList.Items) < 2 {
		log("[rollback] Недостаточно ревизий для отката (нужно минимум 2)")
		return fmt.Errorf("недостаточно ревизий для отката (нужно минимум 2)")
	}

	curRevStr, ok := dep.Annotations["deployment.kubernetes.io/revision"]
	if !ok {
		log("[rollback] Не удалось определить текущую ревизию deployment")
		return fmt.Errorf("не удалось определить текущую ревизию deployment")
	}
	curRev, err := strconv.ParseInt(curRevStr, 10, 64)
	if err != nil {
		log("[rollback] Не удалось преобразовать ревизию deployment")
		return fmt.Errorf("не удалось преобразовать ревизию deployment")
	}

	var targetRS *appsv1.ReplicaSet
	if revision > 0 {
		for _, rs := range rsList.Items {
			if rsRev, ok := rs.Annotations["deployment.kubernetes.io/revision"]; ok {
				if rsRev == fmt.Sprintf("%d", revision) {
					targetRS = &rs
					break
				}
			}
		}
		if targetRS == nil {
			log(fmt.Sprintf("[rollback] Ревизия %d не найдена", revision))
			return fmt.Errorf("ревизия %d не найдена", revision)
		}
		log(fmt.Sprintf("[rollback] Найдена целевая ревизия: %s (реплики: %d)", targetRS.Name, targetRS.Status.Replicas))
	} else {
		targetRevision := curRev - 1
		for _, rs := range rsList.Items {
			if rsRev, ok := rs.Annotations["deployment.kubernetes.io/revision"]; ok {
				rsRevInt, _ := strconv.ParseInt(rsRev, 10, 64)
				if rsRevInt == targetRevision {
					targetRS = &rs
					break
				}
			}
		}
		if targetRS == nil {
			log(fmt.Sprintf("[rollback] Не найдена предыдущая ревизия %d", targetRevision))
			return fmt.Errorf("не найдена предыдущая ревизия %d", targetRevision)
		}
		log(fmt.Sprintf("[rollback] Используем предыдущую ревизию: %s (реплики: %d)", targetRS.Name, targetRS.Status.Replicas))
	}

	log("[rollback] Обновляем deployment с шаблоном пода из выбранного ReplicaSet...")

	// --- Исправление: повторяем Update при конфликте ---
	for attempt := 0; attempt < 5; attempt++ {
		// Получаем актуальный deployment
		dep, err = c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			log(fmt.Sprintf("[rollback] Ошибка получения deployment перед Update: %v", err))
			return fmt.Errorf("ошибка получения deployment перед Update: %v", err)
		}
		dep.Spec.Template = targetRS.Spec.Template
		_, err = c.clientset.AppsV1().Deployments(namespace).Update(ctx, dep, metav1.UpdateOptions{})
		if err == nil {
			break
		}
		if strings.Contains(err.Error(), "the object has been modified") {
			log("[rollback] Конфликт версий deployment, пробуем ещё раз...")
			time.Sleep(1 * time.Second)
			continue
		}
		log(fmt.Sprintf("[rollback] Ошибка обновления deployment: %v", err))
		return fmt.Errorf("ошибка обновления deployment: %v", err)
	}

	log("[rollback] Ожидание завершения отката...")
	err = wait.PollUntilContextCancel(ctx, 2*time.Second, true, func(ctx context.Context) (bool, error) {
		dep, err := c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			log(fmt.Sprintf("[rollback] Ошибка чтения deployment: %v", err))
			return false, err
		}

		ready := dep.Status.AvailableReplicas == *dep.Spec.Replicas &&
			dep.Status.UpdatedReplicas == *dep.Spec.Replicas &&
			dep.Status.Replicas == *dep.Spec.Replicas &&
			dep.Status.UnavailableReplicas == 0

		log(fmt.Sprintf("[rollback] Статус: доступно %d/%d, обновлено: %d, всего: %d, недоступно: %d", dep.Status.AvailableReplicas, *dep.Spec.Replicas, dep.Status.UpdatedReplicas, dep.Status.Replicas, dep.Status.UnavailableReplicas))

		return ready, nil
	})

	if err != nil {
		log(fmt.Sprintf("[rollback] ОШИБКА ожидания завершения отката: %v", err))
		return fmt.Errorf("ошибка ожидания завершения отката: %v", err)
	}

	log("[rollback] Откат deployment успешно завершен")
	return nil
}

func (c *K8sClient) RestartDeploymentWithLogs(ctx context.Context, namespace, name string, logCh chan<- string) error {
	if c.clientset == nil {
		return fmt.Errorf("client not initialized")
	}

	log := func(msg string) {
		if logCh != nil {
			logCh <- msg
		}
	}

	dep, err := c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		log(fmt.Sprintf("Ошибка получения Deployment: %v", err))
		return err
	}

	if dep.Spec.Template.Annotations == nil {
		dep.Spec.Template.Annotations = map[string]string{}
	}

	dep.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339Nano)

	_, err = c.clientset.AppsV1().Deployments(namespace).Update(ctx, dep, metav1.UpdateOptions{})
	if err != nil {
		log(fmt.Sprintf("Ошибка обновления Deployment: %v", err))
		return err
	}

	log("🚀 Rollout restart запущен...")

	return wait.PollUntilContextCancel(ctx, 2*time.Second, true, func(ctx context.Context) (bool, error) {
		updated, err := c.clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			log(fmt.Sprintf("Ошибка чтения Deployment: %v", err))
			return false, err
		}

		ready := updated.Generation <= updated.Status.ObservedGeneration &&
			updated.Status.UpdatedReplicas == *updated.Spec.Replicas &&
			updated.Status.AvailableReplicas == *updated.Spec.Replicas &&
			updated.Status.UnavailableReplicas == 0

		log(fmt.Sprintf(
			"🌀 Прогресс: обновлено %d/%d, готово %d",
			updated.Status.UpdatedReplicas,
			*updated.Spec.Replicas,
			updated.Status.AvailableReplicas,
		))

		if ready {
			log("✅ Rollout завершён успешно.")
			return true, nil
		}

		return false, nil
	})
}

// GetClientset возвращает клиент kubernetes
func (c *K8sClient) GetClientset() kubernetes.Interface {
	return c.clientset
}

// ListAvailableRevisions возвращает список всех ревизий (ReplicaSet) для отката Deployment
func (c *K8sClient) ListAvailableRevisions(ctx context.Context, namespace, deploymentName string) ([]RevisionInfo, error) {
	dep, err := c.GetClientset().AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	selector, err := metav1.LabelSelectorAsSelector(dep.Spec.Selector)
	if err != nil {
		return nil, err
	}
	rsList, err := c.GetClientset().AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector.String(),
	})
	if err != nil {
		return nil, err
	}

	var revisions []RevisionInfo
	for _, rs := range rsList.Items {
		revStr := rs.Annotations["deployment.kubernetes.io/revision"]
		if revStr == "" {
			continue
		}
		var rev int64
		fmt.Sscanf(revStr, "%d", &rev)
		image := ""
		if len(rs.Spec.Template.Spec.Containers) > 0 {
			image = rs.Spec.Template.Spec.Containers[0].Image
		}
		revisions = append(revisions, RevisionInfo{
			Revision: rev,
			RSName:   rs.Name,
			Image:    image,
		})
	}
	// Сортируем по ревизии по возрастанию
	sort.Slice(revisions, func(i, j int) bool {
		return revisions[i].Revision < revisions[j].Revision
	})
	return revisions, nil
}

// GetPodLogs возвращает логи пода
func (c *K8sClient) GetPodLogs(ctx context.Context, namespace, podName string, opts *PodLogsOptions) (string, error) {
	if c.clientset == nil {
		return "", fmt.Errorf("client not initialized")
	}

	podLogOptions := &corev1.PodLogOptions{}
	if opts != nil {
		podLogOptions.Previous = opts.Previous
		podLogOptions.TailLines = &opts.TailLines
		podLogOptions.SinceSeconds = &opts.SinceSeconds
		podLogOptions.Timestamps = opts.Timestamps
	}

	logs, err := c.clientset.CoreV1().Pods(namespace).GetLogs(podName, podLogOptions).DoRaw(ctx)
	if err != nil {
		return "", fmt.Errorf("ошибка получения логов пода %s: %v", podName, err)
	}

	return string(logs), nil
}
