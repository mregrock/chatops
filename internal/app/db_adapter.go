package app

import (
	"db/models"
	"db/repository"
)

type DBAdapter struct{}

func (a *DBAdapter) GetDutyUsersByLabel(label string) ([]models.User, error) {
	return repository.GetDutyUsersByLabel(label)
}
