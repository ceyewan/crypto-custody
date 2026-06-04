package model

import "gorm.io/gorm"

// BackupRecord 记录离线端热备份、冷备份和恢复历史。
type BackupRecord struct {
	gorm.Model
	BackupNo   string `gorm:"column:backup_no;uniqueIndex;size:80;not null;comment:备份编号"`
	BackupType string `gorm:"column:backup_type;size:20;index;comment:hot/cold"`
	FileName   string `gorm:"column:file_name;size:255;comment:文件名"`
	FilePath   string `gorm:"column:file_path;size:500;comment:文件路径"`
	FileHash   string `gorm:"column:file_hash;size:128;comment:文件哈希"`
	Encrypted  bool   `gorm:"column:encrypted;default:false;comment:是否加密"`
	CreatedBy  string `gorm:"column:created_by;size:80;comment:创建人"`
	RestoredBy string `gorm:"column:restored_by;size:80;comment:恢复人"`
	RestoredAt *int64 `gorm:"column:restored_at;comment:恢复时间Unix秒"`
	Status     string `gorm:"column:status;size:30;index;default:'created';comment:created/restored/failed"`
}
