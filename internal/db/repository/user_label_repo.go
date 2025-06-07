package repository

import (
	"chatops/internal/db/config"
	"chatops/internal/db/models"
)

// CreateUserLabel создает новую метку для пользователя
func CreateUserLabel(userID uint, label string) (*models.UserLabel, error) {
	userLabel := &models.UserLabel{
		User:  models.User{ID: userID},
		Label: label,
	}
	err := config.DB.Create(userLabel).Error
	return userLabel, err
}

// GetUserLabels получает все метки пользователя
func GetUserLabels(userID uint) ([]models.UserLabel, error) {
	var labels []models.UserLabel
	err := config.DB.Preload("User").Where("user_id = ?", userID).Find(&labels).Error
	return labels, err
}

// DeleteUserLabel удаляет метку пользователя
func DeleteUserLabel(labelID uint) error {
	return config.DB.Delete(&models.UserLabel{}, labelID).Error
}

// UpdateUserLabel обновляет текст метки
func UpdateUserLabel(labelID uint, newLabel string) error {
	return config.DB.Model(&models.UserLabel{}).
		Where("id = ?", labelID).
		Update("label", newLabel).Error
}

// GetLabelByName находит метку по имени
func GetLabelByName(label string) (*models.UserLabel, error) {
	var userLabel models.UserLabel
	err := config.DB.Where("label = ?", label).First(&userLabel).Error
	return &userLabel, err
}

// GetDutyUsersByLabel получает всех дежурных пользователей с определенной меткой
func GetDutyUsersByLabel(label string) ([]models.User, error) {
	var users []models.User
	err := config.DB.Joins("JOIN user_labels ON user_labels.user_id = users.id").
		Where("user_labels.label = ? AND users.is_duty = ?", label, true).
		Find(&users).Error
	return users, err
}

// GetDutyUsersWithLabels получает всех дежурных пользователей вместе с их метками
func GetDutyUsersWithLabels() (map[models.User][]string, error) {
	var userLabels []struct {
		models.User
		Label string
	}

	err := config.DB.Table("users").
		Select("users.*, user_labels.label").
		Joins("JOIN user_labels ON user_labels.user_id = users.id").
		Where("users.is_duty = ?", true).
		Scan(&userLabels).Error

	if err != nil {
		return nil, err
	}

	// Группируем результаты по пользователям
	result := make(map[models.User][]string)
	for _, ul := range userLabels {
		user := ul.User
		result[user] = append(result[user], ul.Label)
	}

	return result, nil
}
