// Package storage 提供对系统持久化数据的存储和访问管理
package storage

import (
	"log"
	"sync"

	"gorm.io/gorm"

	"offline-server/storage/db"
	"offline-server/storage/model"
)

// KeyGenStorage 提供对密钥生成会话的存储和访问
type KeyGenStorage struct {
	mu sync.RWMutex // 使用读写锁提高并发效率
}

var (
	keyGenInstance *KeyGenStorage
	keyGenOnce     sync.Once
)

// GetKeyGenStorage 返回 KeyGenStorage 的单例实例
func GetKeyGenStorage() IKeyGenStorage {
	keyGenOnce.Do(func() {
		keyGenInstance = &KeyGenStorage{}
	})
	return keyGenInstance
}

// CreateSession 创建新的密钥生成会话
func (s *KeyGenStorage) CreateSession(keyID, initiatorID string, threshold, totalParts int, participants []string) error {
	if keyID == "" || initiatorID == "" || threshold <= 0 || totalParts <= 0 || len(participants) == 0 {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	// 检查会话是否已存在
	var count int64
	if err := database.Model(&model.KeyGenSession{}).Where("key_id = ?", keyID).Count(&count).Error; err != nil {
		log.Printf("查询密钥生成会话失败: %v", err)
		return err
	}
	if count > 0 {
		return ErrSessionExists
	}

	// 创建新会话
	session := model.KeyGenSession{
		KeyID:        keyID,
		InitiatorID:  initiatorID,
		Threshold:    threshold,
		TotalParts:   totalParts,
		Participants: model.StringSlice(participants),
		Responses:    model.StringMap{},
		Completed:    model.StringMap{},
		Status:       model.StatusCreated,
	}

	if err := database.Create(&session).Error; err != nil {
		log.Printf("创建密钥生成会话失败: %v", err)
		return err
	}

	return nil
}

// GetSession 获取指定密钥ID的生成会话
func (s *KeyGenStorage) GetSession(keyID string) (*model.KeyGenSession, error) {
	if keyID == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var session model.KeyGenSession
	if err := database.Where("key_id = ?", keyID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrSessionNotFound
		}
		log.Printf("获取密钥生成会话失败: %v", err)
		return nil, err
	}
	return &session, nil
}

// UpdateStatus 更新密钥生成会话状态
func (s *KeyGenStorage) UpdateStatus(keyID string, status model.SessionStatus) error {
	if keyID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	result := database.Model(&model.KeyGenSession{}).Where("key_id = ?", keyID).Update("status", status)
	if result.Error != nil {
		log.Printf("更新密钥生成会话状态失败: %v", result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// UpdateResponse 更新参与者对会话的响应状态
func (s *KeyGenStorage) UpdateResponse(keyID, userID string, agreed bool) error {
	if keyID == "" || userID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	// 获取当前会话
	var session model.KeyGenSession
	if err := database.Where("key_id = ?", keyID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrSessionNotFound
		}
		log.Printf("获取密钥生成会话失败: %v", err)
		return err
	}

	// 更新响应
	responses := session.Responses
	if responses == nil {
		responses = model.StringMap{}
	}
	responses[userID] = agreed

	result := database.Model(&model.KeyGenSession{}).Where("key_id = ?", keyID).Update("responses", responses)
	if result.Error != nil {
		log.Printf("更新参与者响应失败: %v", result.Error)
		return result.Error
	}

	return nil
}

// UpdateCompleted 更新参与者完成状态
func (s *KeyGenStorage) UpdateCompleted(keyID, userID string, completed bool) error {
	if keyID == "" || userID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	// 获取当前会话
	var session model.KeyGenSession
	if err := database.Where("key_id = ?", keyID).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrSessionNotFound
		}
		log.Printf("获取密钥生成会话失败: %v", err)
		return err
	}

	// 更新完成状态
	completedMap := session.Completed
	if completedMap == nil {
		completedMap = model.StringMap{}
	}
	completedMap[userID] = completed

	result := database.Model(&model.KeyGenSession{}).Where("key_id = ?", keyID).Update("completed", completedMap)
	if result.Error != nil {
		log.Printf("更新参与者完成状态失败: %v", result.Error)
		return result.Error
	}

	return nil
}

// UpdateAccountAddr 更新会话关联的账户地址
func (s *KeyGenStorage) UpdateAccountAddr(keyID, accountAddr string) error {
	if keyID == "" || accountAddr == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	result := database.Model(&model.KeyGenSession{}).Where("key_id = ?", keyID).Update("account_addr", accountAddr)
	if result.Error != nil {
		log.Printf("更新账户地址失败: %v", result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// DeleteSession 删除指定密钥ID的生成会话
func (s *KeyGenStorage) DeleteSession(keyID string) error {
	if keyID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	result := database.Where("key_id = ?", keyID).Delete(&model.KeyGenSession{})
	if result.Error != nil {
		log.Printf("删除密钥生成会话失败: %v", result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}
