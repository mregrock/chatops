package bot

type Handler struct {
	// TODO: добавить зависимость на ядро приложения (app) или его сервисы
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) HandleUpdate(/* update tgbotapi.Update */) {
	// TODO: распарсить команду и вызвать соответствующий сервис
} 