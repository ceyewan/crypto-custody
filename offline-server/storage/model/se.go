package model

import "gorm.io/gorm"

// SE 表示一个安全芯片，包含外部贴着的用户可读的 ID 和芯片的唯一标识符 CPIC
// 执行安全芯片相关操作时，使用 CPIC 作为唯一标识符
// 但在用户界面上，使用 SE ID 作为用户可读的标识符
type Se struct {
	gorm.Model
	SeId string `gorm:"column:se_id;uniqueIndex;size:10;comment:SE ID"`
	CPIC string `gorm:"column:cpic;uniqueIndex;size:100;comment:SE CPIC"`
}
