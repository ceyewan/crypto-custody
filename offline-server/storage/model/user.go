package model

import "gorm.io/gorm"

// 定义用户的 Role 类型
type Role string

const (
	Admin       Role = "admin"
	Coordinator Role = "coordinator"
	Participant Role = "participant"
	Guest       Role = "guest"
)

// User 用户模型
type User struct {
	gorm.Model
	Username string `gorm:"column:username;uniqueIndex;size:50;comment:用户名"`
	Password string `gorm:"column:password;size:200;comment:加密后的密码"`
	Email    string `gorm:"column:email;size:100;comment:用户邮箱"`
	Role     string `gorm:"column:role;comment:用户角色"`
}
