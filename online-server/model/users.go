package model

import "gorm.io/gorm"

// Role 定义用户的角色类型
type Role string

const (
	RoleAdmin   Role = "admin"   // 管理员
	RoleOfficer Role = "officer" // 警员
	RoleGuest   Role = "guest"   // 游客
)

// User 表示用户模型，包含用户的基本信息和角色。
// 用户通过用户名 (Username)、密码 (Password) 和邮箱 (Email) 唯一标识。
// Role 字段表示用户的角色类型，用于权限管理。
type User struct {
	gorm.Model
	Username string `gorm:"column:username;uniqueIndex;size:50;not null;comment:用户名，唯一标识用户"` // 用户名
	Password string `gorm:"column:password;size:200;not null;comment:加密后的密码"`                // 加密后的密码
	Email    string `gorm:"column:email;uniqueIndex;size:100;not null;comment:用户邮箱，唯一标识"`    // 用户邮箱
	Role     Role   `gorm:"column:role;type:varchar(20);not null;comment:用户角色"`              // 用户角色
}
