// model/wallet.go
package model

import (
	"gorm.io/gorm"
	"time"
)
type Wallet struct {
    gorm.Model
    Address    string `gorm:"uniqueIndex;not null"`
    CaseID     uint   `gorm:"not null"`
    Case       Case   `gorm:"foreignKey:CaseID"`
    Status     string `gorm:"not null"` // active, frozen
    Balance    string // 最后一次查询的余额
    UpdatedAt  time.Time
}