package model

import "gorm.io/gorm"

// Account 代表虚拟货币账户
type Account struct {
	gorm.Model
	Address     string `gorm:"unique;not null;comment:虚拟货币地址"`
	CoinType    string `gorm:"not null;comment:币种类型"` // 如 ETH, BTC 等
	Balance     string `gorm:"not null;comment:当前余额"` // 以字符串形式存储余额，避免精度问题
	ImportedBy  string `gorm:"comment:导入用户名"`         // 记录哪个用户导入了这个账户
	Description string `gorm:"comment:账户备注"`          // 账户的备注说明
}
