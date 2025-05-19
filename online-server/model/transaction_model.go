package model

import (
	"gorm.io/gorm"
)

// TransactionStatus 表示交易的状态
type TransactionStatus int

const (
	StatusPending   TransactionStatus = iota // 交易已创建但未签名
	StatusSigned                             // 交易已签名但未提交
	StatusSubmitted                          // 交易已提交到网络
	StatusConfirmed                          // 交易已被确认
	StatusFailed                             // 交易失败
)

// Transaction 存储以太坊交易的核心信息
type Transaction struct {
	gorm.Model                    // 包含 ID, CreatedAt, UpdatedAt, DeletedAt
	FromAddress string            `gorm:"index;not null"` // 发送方地址
	ToAddress   string            `gorm:"index;not null"` // 接收方地址
	Value       string            `gorm:"not null"`       // 交易金额 (例如 "1.5 ETH")
	MessageHash string            `gorm:"index"`          // 消息哈希 (例如交易数据的哈希)
	TxHash      string            `gorm:"index;unique"`   // 交易哈希 (Transaction Hash)
	Signature   []byte            `gorm:"type:blob"`      // 交易签名
	Receipt     []byte            `gorm:"type:blob"`      // 交易回执 (例如 JSON 序列化后的回执)
	Status      TransactionStatus `gorm:"not null"`       // 交易状态
}
