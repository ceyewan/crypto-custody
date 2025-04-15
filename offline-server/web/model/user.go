package model

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Username string `gorm:"uniqueIndex;size:50"`
	Password string `gorm:"size:200"`
	Email    string `gorm:"size:100"`
	Role     string `gorm:"user_roles;"`
}
