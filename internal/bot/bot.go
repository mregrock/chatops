package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"
	"chatops/internal/monitoring/client.go"
	"github.com/joho/godotenv"
	telebot "gopkg.in/telebot.v3"
)









func revisionsHandler(c telebot.Context) error {
	// TODO: Реализовать логику для команды revisions
	return c.Send("Выполняется команда revisions...")
}






type handlerFunc func(telebot.Context) error

func withConfirmation(handler handlerFunc) handlerFunc {
	return func(c telebot.Context) error {
		yesBtn := telebot.InlineButton{
			Unique: "confirm_yes",
			Text:   "Да",
		}
		noBtn := telebot.InlineButton{
			Unique: "confirm_no",
			Text:   "Нет",
		}

		_, err := c.Bot().Send(c.Chat(), "Вы уверены?", &telebot.ReplyMarkup{
			InlineKeyboard: [][]telebot.InlineButton{
				{yesBtn, noBtn},
			},
		})

		if err != nil {
			return err
		}

		c.Bot().Handle(&yesBtn, func(cb telebot.Context) error {
			if cb.Sender().ID != c.Sender().ID {
				return cb.Respond(&telebot.CallbackResponse{Text: "Это не для вас"})
			}
			cb.Respond()
			return handler(cb)
		})

		c.Bot().Handle(&noBtn, func(cb telebot.Context) error {
			if cb.Sender().ID != c.Sender().ID {
				return cb.Respond(&telebot.CallbackResponse{Text: "Это не для вас"})
			}
			cb.Respond()
			return cb.Send("Отмена")
		})
		return nil
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Не удалось загрузить .env файл")
	}

	token := os.Getenv("TELEGRAM_API")
	if token == "" {
		log.Fatal("TELEGRAM_API не найден в .env")
	}

	pref := telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := telebot.NewBot(pref)
	if err != nil {
		log.Fatal(err)
	}

	helpMsg := `Доступные функции:

	/start - чтобы авторизоваться
	/status [name\id] - ?
	/metric [сервис] [метрика] 
	/list_metric [сервис] [метрика] 
	/scale
	/restart
	/rollback
	/history
	/operations
	/revisions
	/help - выводит все доступные команды`

	var commandHandlers = map[string]handlerFunc{
		"/status":     statusHandler,
		"/metric":     metricHandler,
		"/list_metric":     listMetricHandler,
		"/scale":      scaleHandler,
		"/restart":    restartHadler,
		"/rollback":   rollbackHandler,
		"/history":    historyHandler,
		"/operations": operationsHandler,
		"/revisions":  revisionsHandler,
	}
	var userState = make(map[int64]string)
	var userLogin = make(map[int64]string)
	bot.Handle("/start", func(c telebot.Context) error {
		userID := c.Sender().ID
		if _, ok  := userState[userID]; ok {
			return nil
		}
		userState[userID] = "login"
		return c.Send("Введите свой логин:")
	})
	bot.Handle("/help", func(c telebot.Context) error {
		return c.Send(helpMsg)
	})

	bot.Handle(telebot.OnText, func(c telebot.Context) error {
		text := c.Text()
		userID := c.Sender().ID
		if strings.HasPrefix(text, "/") {
			if _, ok  := userState[userID]; ok {
				return nil
			}
			parts := strings.SplitN(text, " ", 2)
			cmd := parts[0]
			if handler, ok := commandHandlers[cmd]; ok {
				return withConfirmation(handler)(c)
			}
			return c.Send("Введите одну из предложенных команд")
		} else {
			switch userState[userID]  {
			case "login":
				userState[userID] = "password"
				userLogin[userID] = c.Text()
				return c.Send("Теперь введите пароль:")
			case "password":

				// TODO: сделать проверку логина и пароля
				delete(userState, userID)
				delete(userLogin, userID)
				return c.Send("Авторизация успешна!")
			default:
				return c.Send("Непонятные входные данные или что то пошло не так.")
			}
		}

	})
	commands := []telebot.Command{
		{Text: "start", Description: "Авторизация в системе"},
		{Text: "status", Description: "Проверка статуса [name|id]"},
		{Text: "metric", Description: "Получение метрик сервиса"},
		{Text: "list_metric", Description: "Полуение списка доступных "},
		{Text: "scale", Description: "Масштабирование"},
		{Text: "restart", Description: "Перезапуск"},
		{Text: "rollback", Description: "Откат изменений"},
		{Text: "history", Description: "История операций"},
		{Text: "operations", Description: "Список операций"},
		{Text: "revisions", Description: "Список ревизий"},
		
		{Text: "help", Description: "Список доступных команд"},
	}
	if err := bot.SetCommands(commands); err != nil {
		log.Println("Ошибка при установке команд:", err)
	}
	
	bot.Start()
}
