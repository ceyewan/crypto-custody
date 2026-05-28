package storage

import (
	"log"
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
