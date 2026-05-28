package model

import "gorm.io/gorm"

// OfflineTaskStatus 表示离线任务状态。
type OfflineTaskStatus string

const (
	OfflineTaskImported   OfflineTaskStatus = "imported"
	OfflineTaskProcessing OfflineTaskStatus = "processing"
	OfflineTaskCompleted  OfflineTaskStatus = "completed"
	OfflineTaskFailed     OfflineTaskStatus = "failed"
)

// OfflineTask 记录在线/离线 JSON 任务包和结果包的摘要。
type OfflineTask struct {
	gorm.Model
	TaskNo         string            `gorm:"column:task_no;uniqueIndex;size:100;not null;comment:任务编号"`
	TaskType       string            `gorm:"column:task_type;size:64;not null;comment:任务类型"`
	SourceSystem   string            `gorm:"column:source_system;size:64;comment:来源系统"`
	PayloadHash    string            `gorm:"column:payload_hash;size:128;comment:payload hash"`
	ResultHash     string            `gorm:"column:result_hash;size:128;comment:result hash"`
	RawPackagePath string            `gorm:"column:raw_package_path;size:500;comment:原始任务包路径"`
	Status         OfflineTaskStatus `gorm:"column:status;type:varchar(20);not null;default:'imported';comment:任务状态"`
}
