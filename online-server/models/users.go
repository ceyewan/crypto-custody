package models

//import "gorm.io/gorm"

type User struct {
	Userid   int    `gorm:"primaryKey;autoIncrement"`
	Username string `gorm:"unique;not null"`
	Password string `gorm:"not null"`
	Roleid   int    `gorm:"not null"`
}
