package kube

import (
	"context"
	"fmt"
	"sort"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sClientInterface interface {
	GetPodStatus(namespace, podName string) (string, error)
	ScaleDeploymentWithLogs(namespace, name string, replicas int32, logCh chan<- string) error
	RollbackDeploymentWithLogs(namespace, name string, logCh chan<- string) error
	RestartDeploymentWithLogs(namespace, name string, logCh chan<- string) error
	GetClientset() kubernetes.Interface
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

func (c *K8sClient) GetPodStatus(namespace, podName string) (string, error) {
	pod, err := c.clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
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

func (c *K8sClient) ScaleDeploymentWithLogs(namespace, name string, replicas int32, logCh chan<- string) error {
	if c.clientset == nil {
		return fmt.Errorf("client not initialized")
	}

	ctx := context.TODO()

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
	return wait.PollImmediate(2*time.Second, 2*time.Minute, func() (bool, error) {
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

func (c *K8sClient) RollbackDeploymentWithLogs(namespace, name string, logCh chan<- string) error {
	log := func(msg string) {
		if logCh != nil {
			logCh <- msg
		}
	}

	log("[rollback] Получаем deployment...")
	dep, err := c.clientset.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
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
	rsList, err := c.clientset.AppsV1().ReplicaSets(namespace).List(context.TODO(), metav1.ListOptions{
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

	// Сортируем ReplicaSets по времени создания (от новых к старым)
	sort.Slice(rsList.Items, func(i, j int) bool {
		return rsList.Items[i].CreationTimestamp.After(rsList.Items[j].CreationTimestamp.Time)
	})

	if len(rsList.Items) < 2 {
		log("[rollback] Недостаточно ревизий для отката (нужно минимум 2)")
		return fmt.Errorf("недостаточно ревизий для отката (нужно минимум 2)")
	}

	currentRS := rsList.Items[0]
	previousRS := rsList.Items[1]

	log(fmt.Sprintf("[rollback] Текущий ReplicaSet: %s (реплики: %d, ревизия: %s)", currentRS.Name, currentRS.Status.Replicas, currentRS.Annotations["deployment.kubernetes.io/revision"]))
	log(fmt.Sprintf("[rollback] Предыдущий ReplicaSet: %s (реплики: %d, ревизия: %s)", previousRS.Name, previousRS.Status.Replicas, previousRS.Annotations["deployment.kubernetes.io/revision"]))

	log("[rollback] Обновляем deployment с шаблоном пода из предыдущего ReplicaSet...")
	dep.Spec.Template = previousRS.Spec.Template
	_, err = c.clientset.AppsV1().Deployments(namespace).Update(context.TODO(), dep, metav1.UpdateOptions{})
	if err != nil {
		log(fmt.Sprintf("[rollback] Ошибка обновления deployment: %v", err))
		return fmt.Errorf("ошибка обновления deployment: %v", err)
	}

	log("[rollback] Ожидание завершения отката...")
	err = wait.PollImmediate(2*time.Second, 5*time.Minute, func() (bool, error) {
		dep, err := c.clientset.AppsV1().Deployments(namespace).Get(context.TODO(), name, metav1.GetOptions{})
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

func (c *K8sClient) RestartDeploymentWithLogs(namespace, name string, logCh chan<- string) error {
	if c.clientset == nil {
		return fmt.Errorf("client not initialized")
	}

	ctx := context.TODO()

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

	return wait.PollImmediate(2*time.Second, 2*time.Minute, func() (bool, error) {
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
