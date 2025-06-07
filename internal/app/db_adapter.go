package app

import (
	"chatops/internal/db/models"
	"chatops/internal/db/repository"
)

type DBAdapter struct{}

func (a *DBAdapter) GetDutyUsersByLabel(label string) ([]models.User, error) {
	return repository.GetDutyUsersByLabel(label)
}
