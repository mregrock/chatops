package main

import (
	"chatops/internal/db/migrations"
	"log"
)

func main() {
	// Создание таблиц в базе данных
	if err := migrations.AutoMigrate(); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
	log.Println("Database initialized successfully")
}
