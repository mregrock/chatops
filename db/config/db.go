// config/db.go
package config

import (
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() error {
	host := os.Getenv("DB_HOST")
	if host == "" {
		host = "localhost" // по умолчанию для локального запуска
	}

	port := os.Getenv("DB_PORT")
	if port == "" {
		port = "5433" // порт, проброшенный в docker-compose
	}

	user := os.Getenv("POSTGRES_USER")
	if user == "" {
		user = "user1"
	}

	password := os.Getenv("POSTGRES_PASSWORD")
	if password == "" {
		password = "pass1"
	}

	dbname := os.Getenv("POSTGRES_DB")
	if dbname == "" {
		dbname = "db1"
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		host, user, password, dbname, port)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect DB: %v", err)
	}

	DB = db
	return nil
}
