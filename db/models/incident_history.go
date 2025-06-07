package models

import (
	"time"
)

type IncidentHistory struct {
	Time   time.Time `gorm:"not null"`
	ID     uint      `gorm:"primaryKey"`
	Status string    `gorm:"not null"`
}
