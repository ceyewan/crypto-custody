package service

import (
	"fmt"
	"online-server/model"
	"online-server/utils"
)

type CaseService struct{}

func NewCaseService() *CaseService {
	return &CaseService{}
}

func (s *CaseService) List(page, pageSize int, caseNo, keyword, status string) ([]model.Case, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	query := utils.GetDB().Model(&model.Case{})
	if caseNo != "" {
		query = query.Where("case_no LIKE ?", "%"+caseNo+"%")
	}
	if keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询案件总数失败: %w", err)
	}
	var cases []model.Case
	if err := query.Order("created_at DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&cases).Error; err != nil {
		return nil, 0, fmt.Errorf("查询案件列表失败: %w", err)
	}
	return cases, total, nil
}
