package model

import "gorm.io/gorm"

type OfflineTaskType string

const (
	OfflineTaskCustodyKeygen OfflineTaskType = "custody_keygen"
	OfflineTaskSign          OfflineTaskType = "sign"
)

type OfflineTaskStatus string

const (
	OfflineTaskExported  OfflineTaskStatus = "exported"
	OfflineTaskImported  OfflineTaskStatus = "imported"
	OfflineTaskCompleted OfflineTaskStatus = "completed"
	OfflineTaskFailed    OfflineTaskStatus = "failed"
)

// OfflineTask 记录在线端导出给离线端的任务包和导入的结果包摘要。
type OfflineTask struct {
	gorm.Model
	TaskNo        string            `gorm:"column:task_no;uniqueIndex;size:80;not null;comment:任务编号"`
	TaskType      OfflineTaskType   `gorm:"column:task_type;size:40;index;not null;comment:任务类型"`
	CaseID        *uint             `gorm:"column:case_id;index;comment:案件ID"`
	CaseNo        string            `gorm:"column:case_no;size:80;index;comment:案件编号"`
	AccountID     *uint             `gorm:"column:account_id;index;comment:账户ID"`
	TransactionID *uint             `gorm:"column:transaction_id;index;comment:交易ID"`
	PayloadHash   string            `gorm:"column:payload_hash;size:128;comment:任务包哈希"`
	ResultHash    string            `gorm:"column:result_hash;size:128;comment:结果包哈希"`
	Status        OfflineTaskStatus `gorm:"column:status;size:30;index;default:'exported';comment:任务状态"`
	ExportedBy    string            `gorm:"column:exported_by;size:80;comment:导出人"`
	ImportedBy    string            `gorm:"column:imported_by;size:80;comment:导入人"`
	ExportedAt    *int64            `gorm:"column:exported_at;comment:导出时间Unix秒"`
	ImportedAt    *int64            `gorm:"column:imported_at;comment:导入时间Unix秒"`
}
