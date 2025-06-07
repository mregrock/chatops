package postgres

import "context"

type Store struct {
	// TODO: добавить подключение к БД (*sql.DB или *pgxpool.Pool)
}

func NewStore() (*Store, error) {
	// TODO: подключиться к БД
	return &Store{}, nil
}

func (s *Store) IsUserAllowed(ctx context.Context, userID int64) (bool, error) {
	// TODO: выполнить SELECT COUNT(1) FROM allowed_users...
	return true, nil
} 