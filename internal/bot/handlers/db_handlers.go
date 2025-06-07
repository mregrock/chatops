package handlers

import (


	telebot "gopkg.in/telebot.v3"
)



// db
func  historyHandler(c telebot.Context) error {
	// TODO: Реализовать логику для команды history
	return c.Send("Выполняется команда history...")
}

// db
func  operationsHandler(c telebot.Context) error {
	// TODO: Реализовать логику для команды operations
	return c.Send("Выполняется команда operations...")
}
