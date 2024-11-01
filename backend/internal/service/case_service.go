// service/case_service.go
package service

import (
	"crypto-custody/internal/model"

	"gorm.io/gorm"
)

type CaseService struct {
	db *gorm.DB
}

func NewCaseService(db *gorm.DB) *CaseService {
	return &CaseService{
		db: db,
	}
}

func (s *CaseService) CreateCase(name, description string, adminID uint) error {
	c := &model.Case{
		Name:        name,
		Description: description,
		AdminID:     adminID,
		Status:      "active",
	}
	return s.db.Create(c).Error
}
