package model

import "gorm.io/gorm"

// AuditLog 是脱敏审计日志。
type AuditLog struct {
	gorm.Model
	Username          string `gorm:"column:username;index;size:100;comment:操作人"`
	Role              string `gorm:"column:role;size:32;comment:操作角色"`
	Action            string `gorm:"column:action;index;size:100;not null;comment:操作动作"`
	ResourceType      string `gorm:"column:resource_type;size:64;comment:资源类型"`
	ResourceID        string `gorm:"column:resource_id;index;size:100;comment:资源编号"`
	Result            string `gorm:"column:result;size:32;not null;comment:success/failure"`
	ErrorMessage      string `gorm:"column:error_message;type:text;comment:错误信息"`
	RedactedDetail    string `gorm:"column:redacted_detail;type:text;comment:脱敏详情"`
	SensitiveRedacted bool   `gorm:"column:sensitive_redacted;not null;default:true;comment:是否已脱敏"`
}
