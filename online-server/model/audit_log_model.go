package model

import "gorm.io/gorm"

// AuditLog 记录在线端关键业务操作。
type AuditLog struct {
	gorm.Model
	RequestID    string `gorm:"column:request_id;size:80;index;comment:请求编号"`
	Username     string `gorm:"column:username;size:80;index;comment:操作用户"`
	Role         string `gorm:"column:role;size:40;comment:操作角色"`
	Action       string `gorm:"column:action;size:80;index;not null;comment:操作动作"`
	ResourceType string `gorm:"column:resource_type;size:80;index;comment:资源类型"`
	ResourceID   string `gorm:"column:resource_id;size:80;comment:资源ID"`
	CaseNo       string `gorm:"column:case_no;size:80;index;comment:案件编号"`
	IP           string `gorm:"column:ip;size:80;comment:客户端IP"`
	UserAgent    string `gorm:"column:user_agent;size:255;comment:客户端标识"`
	BeforeData   string `gorm:"column:before_data;type:text;comment:操作前数据摘要"`
	AfterData    string `gorm:"column:after_data;type:text;comment:操作后数据摘要"`
	Result       string `gorm:"column:result;size:20;index;comment:success/failure"`
	ErrorMessage string `gorm:"column:error_message;type:text;comment:错误信息"`
	LatencyMS    int64  `gorm:"column:latency_ms;comment:响应耗时毫秒"`
}
