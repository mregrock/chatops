package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	telebot "gopkg.in/telebot.v3"
)

// State constants for FSM
const (
	StateInitial = "initial"
	StateLogin   = "login"
	StatePassword = "password"
)

// UserState stores the current state and data for each user
type UserState struct {
	State string
	Login string
}

// Global map to store user states
var userStates = make(map[int64]*UserState)

func getUserState(userID int64) *UserState {
	state, exists := userStates[userID]
	if !exists {
		state = &UserState{State: StateInitial}
		userStates[userID] = state
	}
	return state
}

func statusHandle(c telebot.Context) error {
	// TODO: Реализовать логику для команды status
	return c.Send("Выполняется команда status...")
}

func metricHandle(c telebot.Context) error {
	// TODO: Реализовать логику для команды metric
	return c.Send("Выполняется команда metric...")
}

func scaleHandle(c telebot.Context) error {
	// TODO: Реализовать логику для команды scale
	return c.Send("Выполняется команда scale...")
}

func restartHadle(c telebot.Context) error {
	// TODO: Реализовать логику для команды restart
	return c.Send("Выполняется команда restart...")
}

func historyHandle(c telebot.Context) error {
	// TODO: Реализовать логику для команды history
	return c.Send("Выполняется команда history...")
}

func operationsHandle(c telebot.Context) error {
	// TODO: Реализовать логику для команды operations
	return c.Send("Выполняется команда operations...")
}

func revisionsHandle(c telebot.Context) error {
	// TODO: Реализовать логику для команды revisions
	return c.Send("Выполняется команда revisions...")
}

func rollbackHandle(c telebot.Context) error {
	// TODO: Реализовать логику для команды rollback
	return c.Send("Выполняется команда rollback...")
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

func handleMessage(c telebot.Context) error {
	userID := c.Sender().ID
	state := getUserState(userID)
	text := c.Text()

	switch state.State {
	case StateInitial:
		return c.Send("Введите логин:")
	case StateLogin:
		state.Login = text
		state.State = StatePassword
		return c.Send("Введите пароль:")
	case StatePassword:
		// TODO: здесь можно добавить проверку пароля
		state.State = StateInitial
		return c.Send(fmt.Sprintf("Вы успешно авторизовались с логином %s!", state.Login))
	default:
		return c.Send("Неизвестное состояние")
	}
}

func startCommand(c telebot.Context) error {
	userID := c.Sender().ID
	state := getUserState(userID)
	state.State = StateLogin
	return c.Send("Введите логин:")
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

	helpMsg := "Доступные функции :\n\n/start - чтобы авторизоваться \n/status [name\\id] - ? \n/metric \n/scale \n/restart \n/rollback \n/history \n/operations \n/revisions \n/help - выводит все доступные команды "

	var commandHandlers = map[string]handlerFunc{
		"/status":     statusHandle,
		"/metric":     metricHandle,
		"/scale":      scaleHandle,
		"/restart":    restartHadle,
		"/rollback":   rollbackHandle,
		"/history":    historyHandle,
		"/operations": operationsHandle,
		"/revisions":  revisionsHandle,
	}

	bot.Handle("/start", startCommand)
	bot.Handle("/help", func(c telebot.Context) error {
		return c.Send(helpMsg)
	})

	bot.Handle(telebot.OnText, func(c telebot.Context) error {
		text := c.Text()
		
		// Если это команда
		if strings.HasPrefix(text, "/") {
			parts := strings.SplitN(text, " ", 2)
			cmd := parts[0]
			if handler, ok := commandHandlers[cmd]; ok {
				return withConfirmation(handler)(c)
			}
			return c.Send("Введите одну из предложенных команд")
		}

		// Если это не команда, обрабатываем как часть FSM диалога
		return handleMessage(c)
	})

	bot.Start()
}

