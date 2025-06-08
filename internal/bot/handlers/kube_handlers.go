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
func ScaleHandler(c telebot.Context) error {
	parts := strings.SplitN(c.Text(), " ", 2)
	if len(parts) < 2 {
		return c.Send("Неправильное кол-во параметров ")
	}
	data := strings.SplitN(parts[1], "/", 2)
	if len(data) < 2 {
		return c.Send("Ошибка в парсинге namespace/name ")
		return c.Send("Ошибка в парсинге namespace/name ")
	}
	namespace := data[0]
	name := data[1]
	
	num, err := strconv.Atoi(parts[2])
	if err != nil {
		return c.Send("Ошибки при чтении числа реплик ")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	logCh := make(chan string)
	go func() {
        for msg := range logCh {
            c.Send(msg) // Отправляем каждое сообщение сразу
        }
    }()
	err = GlobalKubeClient.ScaleDeploymentWithLogs(ctx, namespace, name, int32(num), logCh)
	if err != nil {
		return c.Send("Ошибка при выполнении команды: %v", err)
	}

	return nil
}

// kube
func RestartHandler(c telebot.Context) error {

	parts := strings.SplitN(c.Text(), " ", 2)
	if len(parts) < 2 {
		return c.Send("Неправильное кол-во параметров ")
	}
	data := strings.SplitN(parts[1], "/", 2)
	if len(data) < 2 {
		return c.Send("Ошибка в парсинге namespace/name ")
	}
	namespace := data[0]
	name := data[1]

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	logCh := make(chan string)
	go func() {
        for msg := range logCh {
            c.Send(msg) // Отправляем каждое сообщение сразу
        }
    }()
	err := GlobalKubeClient.RestartDeploymentWithLogs(ctx, namespace, name, logCh)
	if err != nil {
        return c.Send("Ошибка при выполнении команды: %v", err)
    }

	return nil
}

// kube
func RollbackHandler(c telebot.Context) error {
	parts := strings.SplitN(c.Text(), " ", 3)
	if len(parts) < 3 {
		return c.Send("Неправильное кол-во параметров ")
	}
	data := strings.SplitN(parts[1], "/", 2)
	if len(data) < 2 {
		return c.Send("Ошибка в парсинге namespace/name ")
	}
	namespace := data[0]
	name := data[1]
	num, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return c.Send("Ошибки при чтении числа реплик ")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	logCh := make(chan string)
	go func() {
        for msg := range logCh {
            c.Send(msg) // Отправляем каждое сообщение сразу
        }
    }()
	err = GlobalKubeClient.RollbackDeploymentWithLogs(ctx, namespace, name, num, logCh)
	if err != nil {
		return c.Send("Ошибка при выполнении команды: %v", err)
	}

	return nil

}

func RevisionsHandler(c telebot.Context) error {
	// TODO: Реализовать логику для команды revisions
	return c.Send("Выполняется команда revisions...")
}
