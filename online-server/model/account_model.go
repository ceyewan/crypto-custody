package model

import "gorm.io/gorm"

// Account 代表虚拟货币账户
type Account struct {
	gorm.Model
	Address           string `gorm:"unique;not null;comment:虚拟货币地址"`
	CoinType          string `gorm:"not null;comment:币种类型"`                                // 如 ETH, BTC 等
	AccountType       string `gorm:"size:40;index;default:'seized_original';comment:账户类型"` // seized_original/custody_wallet
	Balance           string `gorm:"not null;default:'0';comment:当前余额"`                    // 以字符串形式存储余额，避免精度问题
	BalanceSource     string `gorm:"size:30;default:'manual';comment:余额来源"`                // chain/manual/test
	CaseID            *uint  `gorm:"index;comment:关联案件ID"`
	CaseNo            string `gorm:"size:80;index;comment:关联案件编号"`
	ImportedBy        string `gorm:"comment:导入用户名"`                         // 记录哪个用户导入了这个账户
	Source            string `gorm:"size:40;default:'manual';comment:导入来源"` // manual/csv/json/offline_result/test
	KeyMaterialHint   string `gorm:"size:40;default:'none';comment:密钥材料提示"` // none/has_private_key/offline_generated/offline_signed
	OfflineRefNo      string `gorm:"size:100;index;comment:离线任务或结果引用编号"`
	Description       string `gorm:"comment:账户备注"` // 账户的备注说明
	LastBalanceSyncAt *int64 `gorm:"comment:最近余额同步时间Unix秒"`
}
