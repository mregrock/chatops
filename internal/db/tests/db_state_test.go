package tests

import (
	"chatops/internal/db/config"
	"chatops/internal/db/migrations"
	"chatops/internal/db/models"
	"fmt"
	"testing"
)

func TestPrintDatabaseState(t *testing.T) {
	// Инициализация подключения к БД
	if err := migrations.AutoMigrate(); err != nil {
		t.Fatal("Failed to connect to database:", err)
	}

	fmt.Println("\n=== Текущее состояние базы данных ===")

	// Получаем всех пользователей
	var users []models.User
	if err := config.DB.Find(&users).Error; err != nil {
		t.Errorf("Ошибка получения пользователей: %v", err)
		return
	}
	fmt.Println("\nПользователи:")
	for _, u := range users {
		fmt.Printf("ID: %d, Login: %s, JobStatus: %s, IsDuty: %v\n",
			u.ID, u.Login, u.JobStatus, u.IsDuty)
	}

	// Получаем все метки
	var labels []models.UserLabel
	if err := config.DB.Find(&labels).Error; err != nil {
		t.Errorf("Ошибка получения меток: %v", err)
		return
	}
	fmt.Println("\nМетки пользователей:")
	for _, l := range labels {
		fmt.Printf("ID: %d, UserID: %d, Label: %s\n",
			l.ID, l.UserID, l.Label)
	}

	// Получаем все операции
	var operations []models.Operation
	if err := config.DB.Find(&operations).Error; err != nil {
		t.Errorf("Ошибка получения операций: %v", err)
		return
	}
	fmt.Println("\nОперации:")
	for _, o := range operations {
		fmt.Printf("ID: %d, Text: %s, Time: %v\n",
			o.ID, o.Text, o.Time)
	}

	// Получаем все инциденты
	var incidents []models.IncidentHistory
	if err := config.DB.Find(&incidents).Error; err != nil {
		t.Errorf("Ошибка получения инцидентов: %v", err)
		return
	}
	fmt.Println("\nИнциденты:")
	for _, i := range incidents {
		fmt.Printf("ID: %d, Status: %s\n",
			i.ID, i.Status)
	}
}
