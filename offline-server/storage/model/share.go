package model

import "gorm.io/gorm"

// EthereumKeyShard 表示以太坊账户的加密私钥分片。
// 每个私钥分片由用户名 (Username) 和以太坊地址 (Address) 唯一标识。
// 私钥分片是经过加密的 base64 编码字符串，长度约为 6KB-7KB。
// 加密密钥存储在由 PCIC 标识的安全芯片中。
// ShardIndex 表示这是第几个私钥分片。
type EthereumKeyShard struct {
	gorm.Model
	Username     string `gorm:"column:username;size:50;comment:用户名，用于唯一标识私钥分片"`
	Address      string `gorm:"column:address;size:100;comment:以太坊账户地址，用于唯一标识私钥分片"`
	ShardIndex   int    `gorm:"column:shard_index;comment:私钥分片的序号"`
	PCIC         string `gorm:"column:pcic;size:100;comment:加密密钥所在的安全芯片标识"`
	PrivateShard string `gorm:"column:private;type:text;comment:Base64 编码的加密私钥分片"`
}
