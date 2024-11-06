// model/wallet.go
package model

import (
	"gorm.io/gorm"
)

type Wallet struct {
	gorm.Model
	Address       string `gorm:"uniqueIndex;not null"`
	CaseID        uint   `gorm:"not null"`
	Case          Case   `gorm:"foreignKey:CaseID"`
	Status        string `gorm:"not null"` // active, frozen
	Balance       string // 最后一次查询的余额
	privateKeyStr string `gorm:"not null"` // 私钥字符串
}
