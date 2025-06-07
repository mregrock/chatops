package models

type UserLabel struct {
	ID     uint   `gorm:"primaryKey"`
	UserID uint   `gorm:"not null"`
	User   User   `gorm:"foreignKey:UserID"`
	Label  string `gorm:"not null"`
}
