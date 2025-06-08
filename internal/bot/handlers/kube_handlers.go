package handlers

import (
	"context"
	"fmt"
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
    fmt.Print("Начат Scale\n")
    
    parts := strings.SplitN(c.Text(), " ", 3)
    if len(parts) < 3 {
        return c.Send("Неправильное кол-во параметров")
    }
    
    data := strings.SplitN(parts[1], "/", 2)
    if len(data) < 2 {
        return c.Send("Ошибка в парсинге namespace/name")
    }
    
    namespace := data[0]
    name := data[1]
    
    num, err := strconv.Atoi(parts[2])
    if err != nil {
        return c.Send("Ошибки при чтении числа реплик")
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
    defer cancel()
    
    logCh := make(chan string)
    
    // Запускаем горутину для чтения логов
    go func() {
        for msg := range logCh {
            if msg != "" {
                // Используем правильный формат для Send
                c.Send(msg)

                fmt.Println(msg + "\n")
            }
        }
    }()
    
    err = GlobalKubeClient.ScaleDeploymentWithLogs(ctx, namespace, name, int32(num), logCh)
    if err != nil {
		str := fmt.Sprintf("Ошибка при выполнении команды: %v", err)
		fmt.Println(str)
       return err
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
        str := fmt.Sprintf("Ошибка при выполнении команды: %v", err)
		fmt.Println(str)
       return err
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
		str := fmt.Sprintf("Ошибка при выполнении команды: %v", err)
		fmt.Println(str)
       return err
	}

	return nil

}

func RevisionsHandler(c telebot.Context) error {
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
	ans, err := GlobalKubeClient.ListAvailableRevisions(ctx, namespace, name)
	if err != nil {
        str := fmt.Sprintf("Ошибка при выполнении команды: %v", err)
		fmt.Println(str)
       return err
    }
	for _, revision := range ans {
		str := fmt.Sprintf("Revision: %d, RSName: %s, Image: %s", revision.Revision, revision.RSName, revision.Image)
		fmt.Println(str)
		c.Send(str)
	}

	return nil

}
