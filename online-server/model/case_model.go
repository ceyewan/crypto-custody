package model

import "gorm.io/gorm"

type CaseStatus string

const (
	CaseStatusActive   CaseStatus = "active"
	CaseStatusClosed   CaseStatus = "closed"
	CaseStatusArchived CaseStatus = "archived"
)

// Case 保存在线端案件主数据。
type Case struct {
	gorm.Model
	CaseNo           string     `gorm:"column:case_no;uniqueIndex;size:80;not null;comment:案件编号"`
	Name             string     `gorm:"column:name;size:120;not null;comment:案件名称"`
	Description      string     `gorm:"column:description;type:text;comment:案件描述"`
	Status           CaseStatus `gorm:"column:status;size:20;default:'active';index;comment:案件状态"`
	CustodyAccountID *uint      `gorm:"column:custody_account_id;comment:案件托管钱包账户ID"`
	CustodyAddress   string     `gorm:"column:custody_address;size:120;index;comment:案件托管钱包地址"`
	CreatedBy        string     `gorm:"column:created_by;size:80;comment:创建人"`
}
