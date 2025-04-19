// Package storage 提供对系统持久化数据的存储和访问管理
package storage

import (
	"log"
	"sync"

	"gorm.io/gorm"

	"offline-server/storage/db"
	"offline-server/storage/model"
)

// SignStorage 提供对签名会话的存储和访问
type SignStorage struct {
	mu sync.RWMutex // 使用读写锁提高并发效率
}

var (
	signInstance *SignStorage
	signOnce     sync.Once
)

// GetSignStorage 返回 SignStorage 的单例实例
func GetSignStorage() ISignStorage {
	signOnce.Do(func() {
		signInstance = &SignStorage{}
	})
	return signInstance
}

// CreateSession 创建新的签名会话
func (s *SignStorage) CreateSession(sessionKey, initiator, data, accountAddr string, participants []string) error {
	if sessionKey == "" || initiator == "" || data == "" || len(participants) == 0 {
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
	if err := database.Model(&model.SignSession{}).Where("session_key = ?", sessionKey).Count(&count).Error; err != nil {
		log.Printf("查询签名会话失败: %v", err)
		return err
	}
	if count > 0 {
		return ErrSessionExists
	}

	// 创建新会话
	session := model.SignSession{
		SessionKey:   sessionKey,
		Initiator:    initiator,
		Data:         data,
		AccountAddr:  accountAddr,
		Participants: model.StringSlice(participants),
		Responses:    model.StringMap{},
		Results:      model.StringStringMap{},
		Status:       model.StatusCreated,
	}

	if err := database.Create(&session).Error; err != nil {
		log.Printf("创建签名会话失败: %v", err)
		return err
	}

	return nil
}

// GetSession 获取指定密钥ID的签名会话
func (s *SignStorage) GetSession(sessionKey string) (*model.SignSession, error) {
	if sessionKey == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var session model.SignSession
	if err := database.Where("session_key = ?", sessionKey).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrSessionNotFound
		}
		log.Printf("获取签名会话失败: %v", err)
		return nil, err
	}
	return &session, nil
}

// UpdateStatus 更新签名会话状态
func (s *SignStorage) UpdateStatus(sessionKey string, status model.SessionStatus) error {
	if sessionKey == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	result := database.Model(&model.SignSession{}).Where("session_key = ?", sessionKey).Update("status", status)
	if result.Error != nil {
		log.Printf("更新签名会话状态失败: %v", result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// UpdateResponse 更新参与者对会话的响应状态
func (s *SignStorage) UpdateResponse(sessionKey, userName string, agreed bool) error {
	if sessionKey == "" || userName == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	// 获取当前会话
	var session model.SignSession
	if err := database.Where("session_key = ?", sessionKey).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrSessionNotFound
		}
		log.Printf("获取签名会话失败: %v", err)
		return err
	}

	// 更新响应
	responses := session.Responses
	if responses == nil {
		responses = model.StringMap{}
	}
	responses[userName] = agreed

	result := database.Model(&model.SignSession{}).Where("session_key = ?", sessionKey).Update("responses", responses)
	if result.Error != nil {
		log.Printf("更新参与者响应失败: %v", result.Error)
		return result.Error
	}

	return nil
}

// UpdateResult 更新参与者的签名结果
func (s *SignStorage) UpdateResult(sessionKey, userName, result string) error {
	if sessionKey == "" || userName == "" || result == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	// 获取当前会话
	var session model.SignSession
	if err := database.Where("session_key = ?", sessionKey).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrSessionNotFound
		}
		log.Printf("获取签名会话失败: %v", err)
		return err
	}

	// 更新结果
	results := session.Results
	if results == nil {
		results = model.StringStringMap{}
	}
	results[userName] = result

	dbResult := database.Model(&model.SignSession{}).Where("session_key = ?", sessionKey).Update("results", results)
	if dbResult.Error != nil {
		log.Printf("更新参与者签名结果失败: %v", dbResult.Error)
		return dbResult.Error
	}

	return nil
}

// UpdateSignature 更新最终签名结果并将状态标记为已完成
func (s *SignStorage) UpdateSignature(sessionKey, signature string) error {
	if sessionKey == "" || signature == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	result := database.Model(&model.SignSession{}).Where("session_key = ?", sessionKey).Updates(map[string]interface{}{
		"signature": signature,
		"status":    model.StatusCompleted,
	})

	if result.Error != nil {
		log.Printf("更新最终签名结果失败: %v", result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// DeleteSession 删除指定密钥ID的签名会话
func (s *SignStorage) DeleteSession(sessionKey string) error {
	if sessionKey == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	result := database.Where("session_key = ?", sessionKey).Delete(&model.SignSession{})
	if result.Error != nil {
		log.Printf("删除签名会话失败: %v", result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}
