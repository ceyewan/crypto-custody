package model

import "gorm.io/gorm"

// ApprovalStatus 表示审批状态。
type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalRejected ApprovalStatus = "rejected"
)

// Approval 记录移交、销毁等敏感操作的审批。
type Approval struct {
	gorm.Model
	ApprovalID  string         `gorm:"column:approval_id;uniqueIndex;size:100;not null;comment:审批编号"`
	Operation   string         `gorm:"column:operation;size:64;not null;comment:操作类型"`
	ResourceID  string         `gorm:"column:resource_id;index;size:100;not null;comment:资源编号"`
	RequestedBy string         `gorm:"column:requested_by;size:100;not null;comment:发起人"`
	ApprovedBy  string         `gorm:"column:approved_by;size:100;comment:审批人"`
	Role        string         `gorm:"column:role;size:32;comment:审批角色"`
	Status      ApprovalStatus `gorm:"column:status;type:varchar(20);not null;default:'pending';comment:审批状态"`
}
