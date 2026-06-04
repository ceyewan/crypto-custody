package model

import "gorm.io/gorm"

// KeyShardStatus 表示密钥分片状态。
type KeyShardStatus string

const (
	KeyShardStatusPending     KeyShardStatus = "pending"
	KeyShardStatusActive      KeyShardStatus = "active"
	KeyShardStatusTransferred KeyShardStatus = "transferred"
	KeyShardStatusDestroying  KeyShardStatus = "destroying"
	KeyShardStatusDestroyed   KeyShardStatus = "destroyed"
	KeyShardStatusFailed      KeyShardStatus = "failed"
)

// BlobType 表示加密材料类型。
type BlobType string

const (
	BlobTypeMPCShare           BlobType = "mpc_share"
	BlobTypeImportedPrivateKey BlobType = "imported_private_key"
)

// KeyShard 表示一个被 SE 中 AES key 保护的离线密钥材料密文。
type KeyShard struct {
	gorm.Model
	ShardID       string         `gorm:"column:shard_id;uniqueIndex;size:100;not null;comment:分片编号"`
	OfflineKeyID  string         `gorm:"column:offline_key_id;index;size:100;not null;comment:离线密钥编号"`
	Username      string         `gorm:"column:username;index;size:100;not null;comment:参与者用户名"`
	Address       string         `gorm:"column:address;index;size:100;not null;comment:钱包地址"`
	ShardIndex    int            `gorm:"column:shard_index;not null;comment:keygen 原始 party index"`
	RecordID      string         `gorm:"column:record_id;index;size:64;not null;comment:SE 记录编号，32字节hex"`
	SeCPLC        string         `gorm:"column:se_cplc;index;size:128;not null;comment:绑定 SE CPLC"`
	EncryptedBlob string         `gorm:"column:encrypted_blob;type:text;not null;comment:AES-GCM 加密后的密钥材料"`
	BlobType      BlobType       `gorm:"column:blob_type;type:varchar(32);not null;default:'mpc_share';comment:密文类型"`
	KeyVersion    int            `gorm:"column:key_version;not null;default:1;comment:密钥版本"`
	Status        KeyShardStatus `gorm:"column:status;type:varchar(20);not null;default:'active';comment:分片状态"`
}
