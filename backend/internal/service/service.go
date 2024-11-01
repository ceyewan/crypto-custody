// internal/service/service.go
package service

import (
	"gorm.io/gorm"
)

type Services struct {
	User   *UserService
	Case   *CaseService
	Wallet *WalletService
	Log    *LogService
}

func NewServices(db *gorm.DB) *Services {
	return &Services{
		User:   NewUserService(db),
		Case:   NewCaseService(db),
		Wallet: NewWalletService(db),
		Log:    NewLogService(db),
	}
}
