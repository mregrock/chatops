package models

import (
	"time"
)

type Operation struct {
	Time time.Time `gorm:"not null"`
	ID   uint      `gorm:"primaryKey"`
	Text string    `gorm:"not null"`
}
