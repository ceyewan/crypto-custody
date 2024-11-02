// internal/service/service.go
package service

import (
	"crypto-custody/internal/pkg/auth"

	"gorm.io/gorm"
)

type Services struct {
	User   *UserService
	Case   *CaseService
	Wallet *WalletService
	Log    *LogService
}

func NewServices(db *gorm.DB, jwtAuth *auth.JWTAuth) *Services {
	return &Services{
		User:   NewUserService(db, jwtAuth),
		Case:   NewCaseService(db),
		Wallet: NewWalletService(db),
		Log:    NewLogService(db),
	}
}
