package model

import "gorm.io/gorm"

// CaseStatus 定义案件的状态类型
type CaseStatus string

const (
	// CaseInProgressing 进行中
	CaseInProgressing CaseStatus = "progressing"
	// CaseCompleted 已完成
	CaseCompleted CaseStatus = "completed"
	// CaseClosed 已关闭
	CaseClosed CaseStatus = "closed"
)

// Case 表示一个案件，包含案件的基本信息和门限签名相关的配置。
// 每个案件有名称、描述、状态、账户地址、门限值和私钥分片总数。
// 通过账户地址可以查询对应的私钥分片进行签名。
type Case struct {
	gorm.Model
	Name        string     `gorm:"column:name;size:100;comment:案件名称"`              // 案件名称
	Description string     `gorm:"column:description;type:text;comment:案件描述"`      // 案件描述
	Status      CaseStatus `gorm:"column:status;size:50;comment:案件状态"`             // 案件状态
	Address     string     `gorm:"column:address;size:100;comment:账户地址（由密钥生成后更新）"` // 账户地址
	Threshold   int        `gorm:"column:threshold;comment:门限签名所需的最小签名数"`          // 门限签名所需的最小签名数
	TotalShards int        `gorm:"column:total_shards;comment:门限签名总分片数"`           // 门限签名总分片数
}
