package storage

import (
	"log"
	"strings"
	"sync"

	"offline-server/storage/db"
	"offline-server/storage/model"
)

type AuditStorage struct {
	mu sync.RWMutex
}

var (
	auditInstance *AuditStorage
	auditOnce     sync.Once
)

func GetAuditStorage() IAuditStorage {
	auditOnce.Do(func() {
		auditInstance = &AuditStorage{}
	})
	return auditInstance
}

func (s *AuditStorage) CreateAuditLog(entry model.AuditLog) error {
	if entry.Action == "" || entry.Result == "" {
		return ErrInvalidParameter
	}
	entry.SensitiveRedacted = true

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}
	if err := database.Create(&entry).Error; err != nil {
		log.Printf("创建审计日志失败: %v", err)
		return ErrOperationFailed
	}
	return nil
}

func (s *AuditStorage) ListAuditLogs(limit int) ([]model.AuditLog, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var entries []model.AuditLog
	if err := database.Order("created_at DESC").Limit(limit).Find(&entries).Error; err != nil {
		log.Printf("查询审计日志失败: %v", err)
		return nil, ErrOperationFailed
	}
	return entries, nil
}

func (s *AuditStorage) SearchAuditLogs(filter AuditLogFilter) ([]model.AuditLog, error) {
	if filter.Limit <= 0 || filter.Limit > 1000 {
		filter.Limit = 100
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	query := database.Model(&model.AuditLog{})
	if !filter.TimeFrom.IsZero() {
		query = query.Where("created_at >= ?", filter.TimeFrom)
	}
	if !filter.TimeTo.IsZero() {
		query = query.Where("created_at <= ?", filter.TimeTo)
	}
	if filter.Username != "" {
		query = query.Where("username LIKE ?", "%"+filter.Username+"%")
	}
	if filter.Role != "" {
		query = query.Where("role = ?", filter.Role)
	}
	if filter.Action != "" {
		query = query.Where("action LIKE ?", "%"+filter.Action+"%")
	}
	if filter.Resource != "" {
		like := "%" + filter.Resource + "%"
		query = query.Where("resource_type LIKE ? OR resource_id LIKE ? OR redacted_detail LIKE ?", like, like, like)
	}
	if filter.CaseNo != "" {
		like := "%" + filter.CaseNo + "%"
		query = query.Where("resource_id LIKE ? OR redacted_detail LIKE ?", like, like)
	}
	if filter.Address != "" {
		like := "%" + strings.ToLower(filter.Address) + "%"
		query = query.Where("LOWER(resource_id) LIKE ? OR LOWER(redacted_detail) LIKE ?", like, like)
	}
	if filter.Result != "" {
		query = query.Where("result = ?", filter.Result)
	}

	var entries []model.AuditLog
	if err := query.Order("created_at DESC").Limit(filter.Limit).Find(&entries).Error; err != nil {
		log.Printf("筛选审计日志失败: %v", err)
		return nil, ErrOperationFailed
	}
	return entries, nil
}
