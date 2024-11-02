// model/user.go
package model

import (
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username string `gorm:"uniqueIndex;not null"`
	Password string `gorm:"not null"`

	Role string `gorm:"not null"` // system_admin, case_admin, audit_admin
}
