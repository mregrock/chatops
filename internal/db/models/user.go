package models

type User struct {
	ID        uint   `gorm:"primaryKey"`
	Login     string `gorm:"not null"`
	Password  string `gorm:"not null"`
	IsDuty    bool   `gorm:"default:false"`
	JobStatus string `gorm:"not null"`
}
