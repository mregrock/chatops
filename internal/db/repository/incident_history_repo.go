package repository

import (
	"chatops/internal/db/config"
	"chatops/internal/db/models"
	"time"
)

func CreateIncident(status string) (*models.IncidentHistory, error) {
	incident := &models.IncidentHistory{
		Time:   time.Now(),
		Status: status,
	}
	err := config.DB.Create(incident).Error
	return incident, err
}

// GetUserIncidents получает все инциденты пользователя
func GetUserIncidents(userID uint) ([]models.IncidentHistory, error) {
	var incidents []models.IncidentHistory
	err := config.DB.Where("user_id = ?", userID).
		Order("created_at desc").
		Find(&incidents).Error
	return incidents, err
}

// GetRecentIncidents получает инциденты за последние 24 часа
func GetRecentIncidents() ([]models.IncidentHistory, error) {
	var incidents []models.IncidentHistory
	dayAgo := time.Now().Add(-24 * time.Hour)

	err := config.DB.Where("time > ?", dayAgo).
		Order("time desc").
		Find(&incidents).Error

	return incidents, err
}

// UpdateIncidentStatus обновляет статус инцидента
func UpdateIncidentStatus(incidentID uint, newStatus string) error {
	return config.DB.Model(&models.IncidentHistory{}).
		Where("id = ?", incidentID).
		Update("status", newStatus).Error
}

// GetIncidentsByStatus получает инциденты по статусу
func GetIncidentsByStatus(status string) ([]models.IncidentHistory, error) {
	var incidents []models.IncidentHistory
	err := config.DB.Where("status = ?", status).
		Order("time desc").
		Find(&incidents).Error
	return incidents, err
}
