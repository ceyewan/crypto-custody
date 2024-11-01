// service/wallet/service.go
package service

import (
	"crypto-custody/internal/model"

	"gorm.io/gorm"
)

type WalletService struct {
	db *gorm.DB
}

func NewWalletService(db *gorm.DB) *WalletService {
	return &WalletService{
		db: db,
	}
}

func (s *WalletService) CreateWallet(address string, caseID uint) error {
	wallet := &model.Wallet{
		Address: address,
		CaseID:  caseID,
		Status:  "active",
	}
	return s.db.Create(wallet).Error
}
