package model

import (
	"time"

	"gorm.io/gorm"
)

// TransactionStatus 表示交易的状态
type TransactionStatus int

const (
	StatusPending TransactionStatus = iota // 交易已创建但未签名
	StatusSigned                           // 交易已签名但未提交
	StatusSubmitted                        // 交易已提交到网络
	StatusConfirmed                        // 交易已被确认
	StatusFailed                           // 交易失败
)

// Transaction 存储以太坊交易信息
type Transaction struct {
	gorm.Model
	FromAddress     string           `gorm:"index;not null"` // 发送方地址
	ToAddress       string           `gorm:"index;not null"` // 接收方地址
	Value           string           `gorm:"not null"`       // 交易金额，存储字符串以保留精度
	Nonce           uint64           `gorm:"not null"`       // 交易的 nonce
	GasLimit        uint64           `gorm:"not null"`       // Gas 限制
	GasPrice        string           `gorm:"not null"`       // Gas 价格，字符串以保留精度
	Data            []byte           `gorm:"type:blob"`      // 交易数据
	TxHash          string           `gorm:"index"`          // 交易哈希
	Status          TransactionStatus `gorm:"not null"`      // 交易状态
	MessageHash     string           `gorm:"index"`          // 待签名的交易消息哈希
	Signature       []byte           `gorm:"type:blob"`      // 交易签名
	BlockNumber     uint64           // 区块高度
	BlockHash       string           // 区块哈希
	Error           string           `gorm:"type:text"` // 错误信息，如果有的话
	SubmittedAt     *time.Time       // 提交时间
	ConfirmedAt     *time.Time       // 确认时间
	LastCheckedAt   *time.Time       // 上次检查状态的时间
	RetryCount      int              // 重试次数
	TransactionJSON []byte           `gorm:"type:blob"` // 序列化的交易对象

	// 添加其他 GORM 索引的标签，以便于查询
	UserID uint `gorm:"index"` // 关联用户ID，如果需要
}

// TransactionSerializer 辅助结构体，用于序列化/反序列化 Transaction
type TransactionSerializer struct {
	FromAddress string
	ToAddress   string
	Value       string
	Nonce       uint64
	GasLimit    uint64
	GasPrice    string
	Data        []byte
	TxHash      string
	Status      TransactionStatus
	MessageHash string
	Signature   []byte
	ChainID     int64
}