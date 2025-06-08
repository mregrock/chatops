package handlers

import (
	"chatops/internal/bot/yandexgpt"
	"fmt"
	"strings"

	telebot "gopkg.in/telebot.v3"
)

func AiHelpHandler(c telebot.Context) error {
	text := c.Message().Text
	parts := strings.SplitN(text, " ", 2)
	if len(parts) != 2 {
		return c.Send("Пожалуйста, укажите текст запроса после /ai_help")
	}

	userQuery := parts[1]
	answer, err := yandexgpt.SendMessage(userQuery)
	if err != nil {
		return c.Send(fmt.Sprintf("Ошибка при обращении к ИИ: %v", err))
	}

	return c.Send(fmt.Sprintf("Ответ ИИ:\n%s", answer))
}
