package tests

import (
	"db/migrations"
	"db/repository"
	"fmt"
	"testing"
	"time"
)

func TestDatabase(t *testing.T) {
	// Создание таблиц в базе данных
	if err := migrations.AutoMigrate(); err != nil {
		t.Fatal("Failed to migrate database:", err)
	}

	t.Run("Users", testUsers)
	t.Run("Labels", testLabels)
	t.Run("Operations", testOperations)
	t.Run("Incidents", testIncidents)
}

func testUsers(t *testing.T) {
	fmt.Println("\n=== Тестирование пользователей ===")

	// Создание пользователя
	user, err := repository.CreateUser("john_doe", "password123", "engineer")
	if err != nil {
		t.Errorf("Ошибка создания пользователя: %v\n", err)
		return
	}
	fmt.Printf("Создан пользователь: %+v\n", user)

	// Обновление статуса дежурства
	err = repository.UpdateUserDutyStatus(user.ID, true)
	if err != nil {
		t.Errorf("Ошибка обновления статуса: %v\n", err)
		return
	}
	fmt.Println("Статус дежурства обновлен")

	// Получение всех пользователей
	users, err := repository.GetAllUsers()
	if err != nil {
		t.Errorf("Ошибка получения пользователей: %v\n", err)
		return
	}
	fmt.Printf("Список пользователей: %+v\n", users)

	// Получение дежурных пользователей
	dutyUsers, err := repository.GetDutyUsers()
	if err != nil {
		t.Errorf("Ошибка получения дежурных: %v\n", err)
		return
	}
	fmt.Printf("Дежурные пользователи: %+v\n", dutyUsers)
}

func testLabels(t *testing.T) {
	fmt.Println("\n=== Тестирование меток ===")

	// Получаем первого пользователя
	users, err := repository.GetAllUsers()
	if err != nil || len(users) == 0 {
		t.Errorf("Нет пользователей для теста: %v\n", err)
		return
	}
	userID := users[0].ID

	// Создание нескольких меток
	labels := []string{"support", "backend", "database"}
	for _, label := range labels {
		_, err := repository.CreateUserLabel(userID, label)
		if err != nil {
			t.Errorf("Ошибка создания метки %s: %v\n", label, err)
			return
		}
	}
	fmt.Println("Метки созданы")

	// Получение меток пользователя
	userLabels, err := repository.GetUserLabels(userID)
	if err != nil {
		t.Errorf("Ошибка получения меток: %v\n", err)
		return
	}
	fmt.Printf("Метки пользователя: %+v\n", userLabels)

	// Получение дежурных пользователей с меткой support
	dutyUsers, err := repository.GetDutyUsersByLabel("support")
	if err != nil {
		t.Errorf("Ошибка получения дежурных по метке: %v\n", err)
		return
	}
	fmt.Printf("Дежурные с меткой support: %+v\n", dutyUsers)

	// Получение всех дежурных с их метками
	dutyUsersWithLabels, err := repository.GetDutyUsersWithLabels()
	if err != nil {
		t.Errorf("Ошибка получения дежурных с метками: %v\n", err)
		return
	}
	fmt.Printf("Все дежурные с их метками: %+v\n", dutyUsersWithLabels)
}

func testOperations(t *testing.T) {
	fmt.Println("\n=== Тестирование операций ===")

	// Создание операции
	err := repository.CreateOperation("test operation executed")
	if err != nil {
		t.Errorf("Ошибка создания операции: %v\n", err)
		return
	}
	fmt.Println("Операция создана")

	// Ждем немного для теста
	time.Sleep(1 * time.Second)

	// Получение недавних операций
	operations, err := repository.GetRecentOperations()
	if err != nil {
		t.Errorf("Ошибка получения операций: %v\n", err)
		return
	}
	fmt.Printf("Недавние операции: %+v\n", operations)
}

func testIncidents(t *testing.T) {
	fmt.Println("\n=== Тестирование инцидентов ===")

	// Создание инцидента
	incident, err := repository.CreateIncident("open")
	if err != nil {
		t.Errorf("Ошибка создания инцидента: %v\n", err)
		return
	}
	fmt.Printf("Создан инцидент: %+v\n", incident)

	// Обновление статуса инцидента
	err = repository.UpdateIncidentStatus(incident.ID, "resolved")
	if err != nil {
		t.Errorf("Ошибка обновления статуса инцидента: %v\n", err)
		return
	}
	fmt.Println("Статус инцидента обновлен")

	// Получение инцидентов по статусу
	incidents, err := repository.GetIncidentsByStatus("resolved")
	if err != nil {
		t.Errorf("Ошибка получения инцидентов: %v\n", err)
		return
	}
	fmt.Printf("Решенные инциденты: %+v\n", incidents)
}
