package handlers

import (
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
	// TODO: Реализовать логику для команды scale
	return c.Send("Выполняется команда scale...")
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
