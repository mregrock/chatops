package app

type App struct {
	// TODO: добавить логгер, конфиг, клиенты к kube, storage, etc.
}

func New() (*App, error) {
	// TODO: реализовать конструктор
	return &App{}, nil
}

func (a *App) Run() error {
	// TODO: запустить Telegram long-polling и другие фоновые задачи
	return nil
} 