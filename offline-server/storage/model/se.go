package model

import "time"

// SeStatus 表示安全芯片的管理状态。
type SeStatus string

const (
	SeStatusActive    SeStatus = "active"
	SeStatusLost      SeStatus = "lost"
	SeStatusDisabled  SeStatus = "disabled"
	SeStatusDestroyed SeStatus = "destroyed"
)

// Se 表示一个由离线系统统一管理的安全芯片。
// 执行芯片校验时使用 CPLC 作为芯片唯一标识；用户界面展示 se_id。
type Se struct {
	ID              uint `gorm:"primarykey"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	SeID            string   `gorm:"column:se_id;uniqueIndex;size:32;not null;comment:SE ID"`
	CPLC            string   `gorm:"column:cplc;uniqueIndex;size:128;not null;comment:SE CPLC"`
	Status          SeStatus `gorm:"column:status;type:varchar(20);not null;default:'active';comment:芯片状态"`
	CustodyLocation string   `gorm:"column:custody_location;size:200;comment:保管位置"`
	RegisteredBy    string   `gorm:"column:registered_by;size:100;comment:登记人"`
}
