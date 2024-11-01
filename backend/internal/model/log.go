// model/log.go
package model

import (
	"gorm.io/gorm"
)

type Log struct {
    gorm.Model
    UserID    uint   `gorm:"not null"`
    Action    string `gorm:"not null"`
    Resource  string `gorm:"not null"`
    Details   string
    IP        string
}