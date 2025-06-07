package migrations

import (
	"db/config"
	"db/models"
)

func AutoMigrate() error {
	if err := config.InitDB(); err != nil {
		return err
	}
	return config.DB.AutoMigrate(&models.User{}, &models.UserAlias{}, &models.IncidentHistory{}, &models.Operation{})
}
