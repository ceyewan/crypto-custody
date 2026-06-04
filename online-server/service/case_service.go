package service

import (
	"fmt"
	"online-server/model"
	"online-server/utils"
	"time"
)

type CaseService struct{}

func NewCaseService() *CaseService {
	return &CaseService{}
}

func (s *CaseService) GenerateCaseNo() (string, error) {
	prefix := fmt.Sprintf("CASE-%s", time.Now().Format("20060102"))
	for i := 1; i <= 999; i++ {
		caseNo := fmt.Sprintf("%s-%03d", prefix, i)
		var count int64
		if err := utils.GetDB().Model(&model.Case{}).Where("case_no = ?", caseNo).Count(&count).Error; err != nil {
			return "", fmt.Errorf("生成案件编号失败: %w", err)
		}
		if count == 0 {
			return caseNo, nil
		}
	}
	return "", fmt.Errorf("当天案件编号已用尽: %s", prefix)
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
