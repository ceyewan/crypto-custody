package model

import "gorm.io/gorm"

// Role 定义用户的角色类型
type Role string

const (
	RoleAdmin   Role = "admin"   // 管理员
	RoleOfficer Role = "officer" // 警员
	RoleAuditor Role = "auditor" // 审计员
)

// UserStatus 定义用户状态
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
)

// User 表示用户模型，包含用户的基本信息和角色。
// Username 当前承载登录标识，可以是手机号、警号或身份证号。Email 是旧字段，
// 新交互不再展示或要求用户填写，保留用于兼容已有数据库约束。
type User struct {
	gorm.Model
	Username string     `gorm:"column:username;uniqueIndex;size:50;not null;comment:登录标识，手机号/警号/身份证号"` // 登录标识
	Nickname string     `gorm:"column:nickname;size:100;comment:昵称或姓名"`                                // 昵称
	Password string     `gorm:"column:password;size:200;not null;comment:加密后的密码"`                      // 加密后的密码
	Email    string     `gorm:"column:email;uniqueIndex;size:100;comment:历史邮箱字段"`                      // 旧字段，不再作为交互输入
	Role     Role       `gorm:"column:role;type:varchar(20);not null;comment:用户角色"`                    // 用户角色
	Status   UserStatus `gorm:"column:status;type:varchar(20);not null;default:'active';comment:用户状态"`
}
