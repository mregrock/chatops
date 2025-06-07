package repository

import (
	"db/config"
	"db/models"
)

// CreateUser создает нового пользователя
func CreateUser(login string, password string, jobStatus string) (*models.User, error) {
	user := &models.User{
		Login:     login,
		Password:  password,
		JobStatus: jobStatus,
		IsDuty:    false,
	}
	err := config.DB.Create(user).Error
	return user, err
}

// GetUserByID получает пользователя по ID
func GetUserByID(id uint) (*models.User, error) {
	var user models.User
	err := config.DB.First(&user, id).Error
	return &user, err
}

// UpdateUserDutyStatus обновляет статус дежурства пользователя
func UpdateUserDutyStatus(userID uint, isDuty bool) error {
	return config.DB.Model(&models.User{}).Where("id = ?", userID).Update("is_duty", isDuty).Error
}

// GetAllUsers получает всех пользователей
func GetAllUsers() ([]models.User, error) {
	var users []models.User
	err := config.DB.Find(&users).Error
	return users, err
}

// GetDutyUsers получает всех дежурных пользователей
func GetDutyUsers() ([]models.User, error) {
	var users []models.User
	err := config.DB.Where("is_duty = ?", true).Find(&users).Error
	return users, err
}

// AuthenticateUser проверяет существование пользователя по логину и паролю
func AuthenticateUser(login, password string) bool {
	var user models.User
	err := config.DB.Where("login = ? AND password = ?", login, password).First(&user).Error
	return err == nil
}
