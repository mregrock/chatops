package handlers

import (
	"chatops/internal/db/config"
	"chatops/internal/db/models"
	"chatops/internal/db/repository"
	"errors"
	"fmt"

	telebot "gopkg.in/telebot.v3"
)

func HistoryHandler(c telebot.Context) error {
	ans := ""
	var incidents []models.IncidentHistory
	if err := config.DB.Find(&incidents).Error; err != nil {
		ans = fmt.Sprintf("Ошибка получения инцидентов: %v", err)
		c.Send(ans)
		return errors.New(ans)
	}
	ans += "История инцидентов:\n"
	for _, i := range incidents {
		ans += fmt.Sprintf("ID: %d, Status: %s\n", i.ID, i.Status)
	}
	return c.Send(ans)
	// TODO: Реализовать логику для команды history
	// return c.Send("Выполняется команда history...")
}

// db
func OperationsHandler(c telebot.Context) error {
	ans := ""
	var operations []models.Operation
	if err := config.DB.Find(&operations).Error; err != nil {
		ans = fmt.Sprintf("Ошибка получения операций: %v", err)
		c.Send(ans)
		return errors.New(ans)
	}
	ans += "Операции:\n"
	for _, o := range operations {
		ans += fmt.Sprintf("ID: %d, Text: %s, Time: %v\n",
			o.ID, o.Text, o.Time)
	}
	return c.Send(ans)
	// TODO: Реализовать логику для команды operations
	// return c.Send("Выполняется команда operations...")
}

func ProofLoginPaswordHandler(login, password string) bool {
	ok := repository.AuthenticateUser(login, password)
	return ok
}
