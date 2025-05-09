package models

import "gorm.io/gorm"

type Account struct {
	gorm.Model
	Address string `gorm:"unique;not null"`
	Balance string `gorm:"not null"`
}
