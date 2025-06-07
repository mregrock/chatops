package repository

import (
	"db/config"
	"db/models"
)

// CreateUserAlias создает новый алиас для пользователя
func CreateUserAlias(userID uint, alias string) (*models.UserAlias, error) {
	userAlias := &models.UserAlias{
		User:  models.User{ID: userID},
		Alias: alias,
	}
	err := config.DB.Create(userAlias).Error
	return userAlias, err
}

// GetUserAliases получает все алиасы пользователя
func GetUserAliases(userID uint) ([]models.UserAlias, error) {
	var aliases []models.UserAlias
	err := config.DB.Where("user_id = ?", userID).Find(&aliases).Error
	return aliases, err
}

// DeleteUserAlias удаляет алиас пользователя
func DeleteUserAlias(aliasID uint) error {
	return config.DB.Delete(&models.UserAlias{}, aliasID).Error
}

// UpdateUserAlias обновляет имя алиаса
func UpdateUserAlias(aliasID uint, newName string) error {
	return config.DB.Model(&models.UserAlias{}).
		Where("id = ?", aliasID).
		Update("name", newName).Error
}

// GetAliasByName находит алиас по имени
func GetAliasByName(name string) (*models.UserAlias, error) {
	var alias models.UserAlias
	err := config.DB.Where("name = ?", name).First(&alias).Error
	return &alias, err
}
