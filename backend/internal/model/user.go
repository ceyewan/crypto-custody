// model/user.go
package model

import (
	"gorm.io/gorm"
)

// User 是用户模型, ID 已经在 gorm.Model 中定义
type User struct {
	gorm.Model
	Username string `gorm:"uniqueIndex;not null"`
	Password string `gorm:"not null"`
	Phone    string `gorm:"not null"`
	Role     string `gorm:"not null"` // system_admin, case_admin, audit_admin, user
}
