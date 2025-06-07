package migrations

import (
	"chatops/internal/db/config"
	"chatops/internal/db/models"
)

func AutoMigrate() error {
	if err := config.InitDB(); err != nil {
		return err
	}
	return config.DB.AutoMigrate(&models.User{}, &models.UserLabel{}, &models.IncidentHistory{}, &models.Operation{})
}
