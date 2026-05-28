package storage

import (
	"log"
	"sync"

	"gorm.io/gorm"

	"offline-server/storage/db"
	"offline-server/storage/model"
)

type OfflineKeyStorage struct {
	mu sync.RWMutex
}

var (
	offlineKeyInstance *OfflineKeyStorage
	offlineKeyOnce     sync.Once
)

func GetOfflineKeyStorage() IOfflineKeyStorage {
	offlineKeyOnce.Do(func() {
		offlineKeyInstance = &OfflineKeyStorage{}
	})
	return offlineKeyInstance
}

func (s *OfflineKeyStorage) CreateOfflineKey(key model.OfflineKey) (*model.OfflineKey, error) {
	if key.OfflineKeyID == "" || key.Address == "" || key.CoinType == "" || key.Algorithm == "" {
		return nil, ErrInvalidParameter
	}
	if key.Status == "" {
		key.Status = model.OfflineKeyStatusActive
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}
	if err := database.Create(&key).Error; err != nil {
		log.Printf("创建离线密钥失败: %v", err)
		return nil, ErrOperationFailed
	}
	return &key, nil
}

func (s *OfflineKeyStorage) GetOfflineKeyByID(offlineKeyID string) (*model.OfflineKey, error) {
	if offlineKeyID == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}
	var key model.OfflineKey
	if err := database.Where("offline_key_id = ?", offlineKeyID).First(&key).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRecordNotFound
		}
		log.Printf("查询离线密钥失败: %v", err)
		return nil, ErrOperationFailed
	}
	return &key, nil
}

func (s *OfflineKeyStorage) GetOfflineKeyByAddress(address string) (*model.OfflineKey, error) {
	if address == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}
	var key model.OfflineKey
	if err := database.Where("address = ?", address).First(&key).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRecordNotFound
		}
		log.Printf("按地址查询离线密钥失败: %v", err)
		return nil, ErrOperationFailed
	}
	return &key, nil
}

func (s *OfflineKeyStorage) GetOfflineKeyByTaskNo(taskNo string) (*model.OfflineKey, error) {
	if taskNo == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}
	var key model.OfflineKey
	if err := database.Where("task_no = ?", taskNo).First(&key).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRecordNotFound
		}
		log.Printf("按任务编号查询离线密钥失败: %v", err)
		return nil, ErrOperationFailed
	}
	return &key, nil
}

func (s *OfflineKeyStorage) UpdateOfflineKeyOwner(offlineKeyID, logicalOwner string) error {
	if offlineKeyID == "" || logicalOwner == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}
	result := database.Model(&model.OfflineKey{}).
		Where("offline_key_id = ?", offlineKeyID).
		Update("logical_owner", logicalOwner)
	if result.Error != nil {
		log.Printf("更新离线密钥归属失败: %v", result.Error)
		return ErrOperationFailed
	}
	if result.RowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil
}

func (s *OfflineKeyStorage) UpdateOfflineKeyStatus(offlineKeyID string, status model.OfflineKeyStatus) error {
	if offlineKeyID == "" || status == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}
	result := database.Model(&model.OfflineKey{}).
		Where("offline_key_id = ?", offlineKeyID).
		Update("status", status)
	if result.Error != nil {
		log.Printf("更新离线密钥状态失败: %v", result.Error)
		return ErrOperationFailed
	}
	if result.RowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil
}
