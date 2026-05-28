package model

import "gorm.io/gorm"

// OfflineKeyStatus 表示离线密钥对象状态。
type OfflineKeyStatus string

const (
	OfflineKeyStatusActive      OfflineKeyStatus = "active"
	OfflineKeyStatusTransferred OfflineKeyStatus = "transferred"
	OfflineKeyStatusDestroyed   OfflineKeyStatus = "destroyed"
)

// Algorithm 表示离线密钥算法。
type Algorithm string

const (
	AlgorithmGG20ECDSASECP256K1 Algorithm = "GG20_ECDSA_SECP256K1"
	AlgorithmThresholdSM2       Algorithm = "THRESHOLD_SM2"
)

// OfflineKey 记录一个托管密钥或导入私钥对象的业务元数据。
type OfflineKey struct {
	gorm.Model
	OfflineKeyID    string           `gorm:"column:offline_key_id;uniqueIndex;size:100;not null;comment:离线密钥编号"`
	TaskNo          string           `gorm:"column:task_no;index;size:100;comment:来源任务编号"`
	CaseNo          string           `gorm:"column:case_no;index;size:100;comment:案件编号"`
	Address         string           `gorm:"column:address;uniqueIndex;size:100;not null;comment:钱包地址"`
	CoinType        string           `gorm:"column:coin_type;size:32;not null;comment:币种"`
	Algorithm       Algorithm        `gorm:"column:algorithm;type:varchar(64);not null;comment:算法"`
	RequiredSigners int              `gorm:"column:required_signers;comment:业务门限人数"`
	TotalParties    int              `gorm:"column:total_parties;comment:总参与方数量"`
	PublicKey       string           `gorm:"column:public_key;type:text;comment:公钥"`
	LogicalOwner    string           `gorm:"column:logical_owner;size:100;comment:业务归属"`
	Status          OfflineKeyStatus `gorm:"column:status;type:varchar(20);not null;default:'active';comment:密钥状态"`
}
