// model/case.go
package model

import (
    "gorm.io/gorm"
)

type Case struct {
    gorm.Model
    Name        string `gorm:"not null"`
    Status      string `gorm:"not null"` // active, closed
    Description string
    AdminID     uint   `gorm:"not null"`
    Admin       User   `gorm:"foreignKey:AdminID"`
}