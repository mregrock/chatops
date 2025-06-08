package handlers

import (
	"context"
	"strconv"
	"strings"
	"time"

	"chatops/internal/kube"
	telebot "gopkg.in/telebot.v3"
)

var GlobalKubeClient *kube.K8sClient

// SetKubeClient sets the global Kubernetes client for handlers
func SetKubeClient(client *kube.K8sClient) {
	GlobalKubeClient = client
}

// kube
func scaleHandler(c telebot.Context) error {
	parts := strings.SplitN(c.Text(), " ", 3)
	if len(parts) < 3 {
		return c.Send("Неправильное кол-во параметров ")
	}
	data := strings.SplitN(parts[1], "/", 2)
	if len(data) < 2 {
		return c.Send("Неправильное кол-во параметров ")
	}
	namespace := data[0]
	name := data[1]
	logCh := make(chan string)
	num, err := strconv.Atoi(parts[2])
	if err != nil {
		return c.Send("Ошибки при чтении числа реплик ")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = GlobalKubeClient.ScaleDeploymentWithLogs(ctx, namespace, name, int32(num), logCh)
	if err != nil {
		return c.Send("Ошибка при выполнении команды scale: %v", err)
	}

	var result string
	for msg := range logCh {
		result += msg
	}

	return c.Send(result)
}

// kube
func restartHandler(c telebot.Context) error {
	// TODO: Реализовать логику для команды restart
	return c.Send("Выполняется команда restart...")
}

// kube
func rollbackHandler(c telebot.Context) error {
	// TODO: Реализовать логику для команды rollback
	return c.Send("Выполняется команда rollback...")
}
