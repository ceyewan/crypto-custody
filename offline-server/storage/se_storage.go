// Package storage 提供对系统持久化数据的存储和访问管理
package storage

import (
	"log"
	"sync"

	"gorm.io/gorm"

	"offline-server/storage/db"
	"offline-server/storage/model"
)

// SeStorage 提供对安全芯片记录的存储和访问。
type SeStorage struct {
	mu sync.RWMutex
}

var (
	seInstance *SeStorage
	seOnce     sync.Once
)

// GetSeStorage 返回 SeStorage 的单例实例。
func GetSeStorage() ISeStorage {
	seOnce.Do(func() {
		seInstance = &SeStorage{}
	})
	return seInstance
}

// CreateSe 创建新的安全芯片记录。
func (s *SeStorage) CreateSe(seID, cplc, custodyLocation, registeredBy string) (*model.Se, error) {
	if seID == "" || cplc == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var count int64
	if err := database.Model(&model.Se{}).Where("se_id = ? OR cplc = ?", seID, cplc).Count(&count).Error; err != nil {
		log.Printf("查询安全芯片失败: %v", err)
		return nil, ErrOperationFailed
	}
	if count > 0 {
		return nil, ErrSeExists
	}

	se := model.Se{
		SeID:            seID,
		CPLC:            cplc,
		Status:          model.SeStatusActive,
		CustodyLocation: custodyLocation,
		RegisteredBy:    registeredBy,
	}
	if err := database.Create(&se).Error; err != nil {
		log.Printf("创建安全芯片记录失败: %v", err)
		return nil, ErrOperationFailed
	}
	return &se, nil
}

// GetSeBySeId 根据安全芯片ID获取记录。
func (s *SeStorage) GetSeBySeId(seID string) (*model.Se, error) {
	if seID == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var se model.Se
	if err := database.Where("se_id = ?", seID).First(&se).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRecordNotFound
		}
		log.Printf("查询安全芯片记录失败: %v", err)
		return nil, ErrOperationFailed
	}
	return &se, nil
}

// GetSeByCPLC 根据 CPLC 获取安全芯片记录。
func (s *SeStorage) GetSeByCPLC(cplc string) (*model.Se, error) {
	if cplc == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var se model.Se
	if err := database.Where("cplc = ?", cplc).First(&se).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRecordNotFound
		}
		log.Printf("查询安全芯片记录失败: %v", err)
		return nil, ErrOperationFailed
	}
	return &se, nil
}

// GetAllSe 获取所有安全芯片记录。
func (s *SeStorage) GetAllSe() ([]model.Se, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var ses []model.Se
	if err := database.Order("se_id ASC").Find(&ses).Error; err != nil {
		log.Printf("查询所有安全芯片记录失败: %v", err)
		return nil, ErrOperationFailed
	}
	return ses, nil
}

// GetActiveSeIds 选取指定数量的可用安全芯片ID。
// 如果可用 SE 数量少于参与方数量，会循环复用已有 SE。SE 记录仍通过不同 record_id 隔离。
func (s *SeStorage) GetActiveSeIds(count int) ([]string, error) {
	if count <= 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var ses []model.Se
	if err := database.
		Where("status = ?", model.SeStatusActive).
		Order("se_id ASC").
		Limit(count).
		Find(&ses).Error; err != nil {
		log.Printf("获取可用安全芯片记录失败: %v", err)
		return nil, ErrOperationFailed
	}

	if len(ses) == 0 {
		return nil, ErrRecordNotFound
	}

	seIDs := make([]string, count)
	for i := 0; i < count; i++ {
		seIDs[i] = ses[i%len(ses)].SeID
	}
	return seIDs, nil
}

// UpdateSeStatus 更新安全芯片状态。
func (s *SeStorage) UpdateSeStatus(seID string, status model.SeStatus) error {
	if seID == "" || status == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	result := database.Model(&model.Se{}).
		Where("se_id = ?", seID).
		Update("status", status)
	if result.Error != nil {
		log.Printf("更新安全芯片状态失败: %v", result.Error)
		return ErrOperationFailed
	}
	if result.RowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil
}
