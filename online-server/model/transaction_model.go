package model

import (
	"gorm.io/gorm"
)

// TransactionStatus 表示交易的状态
type TransactionStatus int

const (
	StatusPending           TransactionStatus = iota // 兼容旧状态：交易已创建但未签名
	StatusSigned                                     // 交易已签名但未提交
	StatusSubmitted                                  // 交易已提交到网络
	StatusConfirmed                                  // 交易已被确认
	StatusFailed                                     // 交易失败
	StatusDraft                                      // 草稿
	StatusSignatureExported                          // 签名任务已导出
	StatusBroadcasted                                // 已广播
	StatusCancelled                                  // 已取消
)

// Transaction 存储以太坊交易的核心信息
type Transaction struct {
	gorm.Model                        // 包含 ID, CreatedAt, UpdatedAt, DeletedAt
	TxNo            string            `gorm:"index;size:80;comment:交易编号"`
	CaseID          *uint             `gorm:"index;comment:案件ID"`
	CaseNo          string            `gorm:"index;size:80;comment:案件编号"`
	TxType          string            `gorm:"index;size:30;default:'withdraw';comment:交易类型"` // collect/withdraw/test
	FromAccountID   *uint             `gorm:"index;comment:发送账户ID"`
	FromAddress     string            `gorm:"index;not null"` // 发送方地址
	ToAddress       string            `gorm:"index;not null"` // 接收方地址
	Value           string            `gorm:"not null"`       // 交易金额 (例如 "1.5 ETH")
	CoinType        string            `gorm:"size:20;default:'ETH';comment:币种"`
	Reason          string            `gorm:"type:text;comment:交易事由"`
	UnsignedPayload []byte            `gorm:"type:blob;comment:待签名交易数据"`
	MessageHash     string            `gorm:"index;unique"` // 消息哈希 (例如交易数据的哈希)
	TxHash          string            `gorm:"index;unique"` // 交易哈希 (Transaction Hash)
	Signature       []byte            `gorm:"type:blob"`    // 交易签名
	Receipt         []byte            `gorm:"type:blob"`    // 交易回执 (例如 JSON 序列化后的回执)
	Status          TransactionStatus `gorm:"not null"`     // 交易状态
	CreatedBy       string            `gorm:"size:80;comment:创建人"`
	ApprovedBy      string            `gorm:"size:80;comment:审批人"`
	ExportedAt      *int64            `gorm:"comment:签名任务导出时间Unix秒"`
	SignedAt        *int64            `gorm:"comment:签名导入时间Unix秒"`
	BroadcastedAt   *int64            `gorm:"comment:广播时间Unix秒"`
	ConfirmedAt     *int64            `gorm:"comment:确认时间Unix秒"`
}
