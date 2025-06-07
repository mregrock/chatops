package repository

import (
	"db/config"
	"db/models"
	"time"
)

// CreateOperation создает новую операцию
func CreateOperation(text string) error {
	operation := &models.Operation{
		Time: time.Now(),
		Text: text,
	}
	return config.DB.Create(operation).Error
}

// GetRecentOperations получает все операции за последние 5 минут
func GetRecentOperations() ([]models.Operation, error) {
	var operations []models.Operation
	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)
	err := config.DB.Where("time > ?", fiveMinutesAgo).
		Order("time desc").
		Find(&operations).Error
	return operations, err
}

// GetUserOperations получает все операции конкретного пользователя
func GetUserOperations(userID uint) ([]models.Operation, error) {
	var operations []models.Operation
	err := config.DB.Where("user_id = ?", userID).
		Order("created_at desc").
		Find(&operations).Error
	return operations, err
}

// GetOperationByID получает операцию по ID
func GetOperationByID(id uint) (*models.Operation, error) {
	var operation models.Operation
	err := config.DB.First(&operation, id).Error
	return &operation, err
}
