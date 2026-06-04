package storage

import (
	"log"
	"sync"

	"offline-server/storage/db"
	"offline-server/storage/model"
)

type ApprovalStorage struct {
	mu sync.RWMutex
}

var (
	approvalInstance *ApprovalStorage
	approvalOnce     sync.Once
)

func GetApprovalStorage() IApprovalStorage {
	approvalOnce.Do(func() {
		approvalInstance = &ApprovalStorage{}
	})
	return approvalInstance
}

func (s *ApprovalStorage) CreateApproval(approval model.Approval) (*model.Approval, error) {
	if approval.ApprovalID == "" || approval.Operation == "" || approval.ResourceID == "" || approval.RequestedBy == "" {
		return nil, ErrInvalidParameter
	}
	if approval.Status == "" {
		approval.Status = model.ApprovalPending
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}
	if err := database.Create(&approval).Error; err != nil {
		log.Printf("创建审批记录失败: %v", err)
		return nil, ErrOperationFailed
	}
	return &approval, nil
}

func (s *ApprovalStorage) ListApprovals(limit int) ([]model.Approval, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var approvals []model.Approval
	if err := database.Order("created_at DESC").Limit(limit).Find(&approvals).Error; err != nil {
		log.Printf("查询审批记录失败: %v", err)
		return nil, ErrOperationFailed
	}
	return approvals, nil
}

func (s *ApprovalStorage) ListApprovalsPage(page, pageSize int) ([]model.Approval, int64, error) {
	page, pageSize = normalizePage(page, pageSize, 100)

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, 0, ErrDatabaseNotInitialized
	}

	query := database.Model(&model.Approval{})
	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Printf("统计审批记录失败: %v", err)
		return nil, 0, ErrOperationFailed
	}

	var approvals []model.Approval
	if err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&approvals).Error; err != nil {
		log.Printf("分页查询审批记录失败: %v", err)
		return nil, 0, ErrOperationFailed
	}
	return approvals, total, nil
}
