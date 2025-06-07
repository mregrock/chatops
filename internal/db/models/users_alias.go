package models

type UserAlias struct {
	ID     uint   `gorm:"primaryKey"`
	UserID uint   `gorm:"not null"`
	User   User   `gorm:"foreignKey:UserID"`
	Alias  string `gorm:"not null"`
}
