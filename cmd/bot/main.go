package main

import (
	"chatops/internal/app"
	"chatops/internal/bot/handlers"
	"chatops/internal/db/migrations"
	"chatops/internal/kube"
	"chatops/internal/monitoring"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	telebot "gopkg.in/telebot.v3"
)

type handlerFunc func(telebot.Context) error

func withConfirmation(handler handlerFunc) handlerFunc {
	return func(originalContext telebot.Context) error {
		yesBtn := telebot.InlineButton{
			Unique: "confirm_yes",
			Text:   "Да",
		}
		noBtn := telebot.InlineButton{
			Unique: "confirm_no",
			Text:   "Нет",
		}

		_, err := originalContext.Bot().Send(originalContext.Chat(), "Вы уверены?", &telebot.ReplyMarkup{
			InlineKeyboard: [][]telebot.InlineButton{
				{yesBtn, noBtn},
			},
		})

		if err != nil {
			return err
		}

		originalContext.Bot().Handle(&yesBtn, func(cb telebot.Context) error {
			if cb.Sender().ID != originalContext.Sender().ID {
				return cb.Respond(&telebot.CallbackResponse{Text: "Это не для вас"})
			}
			cb.Respond()
			return handler(originalContext)
		})

		originalContext.Bot().Handle(&noBtn, func(cb telebot.Context) error {
			if cb.Sender().ID != originalContext.Sender().ID {
				return cb.Respond(&telebot.CallbackResponse{Text: "Это не для вас"})
			}
			cb.Respond()
			return cb.Send("Отмена")
		})
		return nil
	}
}

func startPoller() {
	// Получаем URL'ы из переменных окружения
	prometheusURL := os.Getenv("PROMETHEUS_URL")
	alertmanagerURL := os.Getenv("ALERTMANAGER_URL")

	// Создаем клиент для мониторинга
	monitoringClient, err := monitoring.NewClient(prometheusURL, alertmanagerURL)
	if err != nil {
		log.Printf("Failed to create monitoring client: %v", err)
		return
	}

	// Создаем поллер с интервалом 40 секунд
	poller := app.NewAlertPoller(monitoringClient, 40*time.Second)

	// Создаем канал для обработки сигналов
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Запускаем поллер
	poller.Start()
	log.Println("Alert poller started")

	// Ждем сигнала для завершения
	<-sigChan
	log.Println("Received shutdown signal")

	// Останавливаем поллер
	poller.Stop()
	log.Println("Alert poller stopped")
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Не удалось загрузить .env файл, используются переменные окружения системы")
	}

	migrations.AutoMigrate()

	token := os.Getenv("TELEGRAM_BOT_TOKEN")

	prometheus_url := os.Getenv("PROMETHEUS_URL")
	alertmanager_url := os.Getenv("ALERTMANAGER_URL")

	monitorClient, err := monitoring.NewClient(prometheus_url, alertmanager_url)

	if err != nil {
		log.Fatal(err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Не удалось определить домашнюю директорию:", err)
	}
	kubeconfigPath := filepath.Join(homeDir, ".kube", "config")
	kubeClient, err := kube.InitClientFromKubeconfig(kubeconfigPath)
	if err != nil {
		log.Fatal(err)
	}
	handlers.SetKubeClient(kubeClient)
	handlers.SetMonitorClient(monitorClient)

	pref := telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := telebot.NewBot(pref)
	if err != nil {
		log.Fatal(err)
	}

	// Запускаем поллер в отдельной горутине
	go startPoller()

	helpMsg := `Доступные функции:

	/start - чтобы авторизоваться
	/status [name или id] - проверка статуса сервиса
	/metric [сервис] [строка] - вывод метрики сервиса
	/list_metric [сервис] [строка] - поиск метрики, содержащую данную строку в названии
	/scale [namespace]/[name] [количество реплик] - масштабирование сервиса
	/restart [namespace]/[name] - перезапуск сервиса
	/rollback [namespace]/[name] [номер ревизии] - откат сервиса к указанной ревизии
	/history - вывод истории операций
	/operations - вывод списка операций
	/revisions [namespace/name] - вывод списка ревизий
	/list_pods [namespace]/[name] - вывод списка pod'ов
  /ai_help [строка] - команда для общения с ИИ и преобразования текста в команды
	/alerts - Проверка алертов
	/help - выводит все доступные команды`

	var commandHandlers = map[string]handlerFunc{
		"/status":      handlers.StatusHandler,
		"/metric":      handlers.MetricHandler,
		"/list_metric": handlers.ListMetricsHandler,
		"/scale":       handlers.ScaleHandler,
		"/restart":     handlers.RestartHandler,
		"/rollback":    handlers.RollbackHandler,
		"/history":     handlers.HistoryHandler,
		"/operations":  handlers.OperationsHandler,
		"/list_pods":   handlers.ListPodsHandler,
		"/revisions":   handlers.RevisionsHandler,
		"/ai_help":     handlers.AiHelpHandler,
		"/alerts":      handlers.AlertsHandler,
	}
	var userState = make(map[int64]string)
	var userLogin = ""
	var userPassword = ""
	var userStatusAuthorization = false

	bot.Handle("/start", func(c telebot.Context) error {
		userID := c.Sender().ID
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
			if _, ok := userState[userID]; ok {
				return nil
			}
			if userStatusAuthorization {
				parts := strings.SplitN(text, " ", 2)
				cmd := parts[0]
				if handler, ok := commandHandlers[cmd]; ok {
					return withConfirmation(handler)(c)
				}
				return c.Send("Введите одну из предложенных команд")
			} else {
				return c.Send("Вы не авторизованы. Введите /start для авторизации.")
			}
		} else {
			switch userState[userID] {
			case "login":
				userState[userID] = "password"
				userLogin = c.Text()
				return c.Send("Теперь введите пароль:")
			case "password":
				delete(userState, userID)
				userPassword = c.Text()
				userStatusAuthorization = handlers.ProofLoginPaswordHandler(userLogin, userPassword)
				if userStatusAuthorization {
					return c.Send("Авторизация успешна!")
				} else {
					return c.Send("Неверный логин или пароль.")
				}
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
		{Text: "list_pods", Description: "Список pod'ов"},
		{Text: "help", Description: "Список доступных команд"},
		{Text: "ai_help", Description: "преобразования текста в команды с помошью ИИ"},
		{Text: "alerts", Description: "Проверка алертов"},
	}
	if err := bot.SetCommands(commands); err != nil {
		log.Println("Ошибка при установке команд:", err)
	}

	bot.Start()
}
