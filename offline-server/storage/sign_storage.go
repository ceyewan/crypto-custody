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
// 通过单例模式确保整个应用程序中只有一个存储实例
func GetSignStorage() ISignStorage {
	signOnce.Do(func() {
		signInstance = &SignStorage{}
	})
	return signInstance
}

// CreateSession 创建新的签名会话
// 参数：
//   - sessionKey: 会话密钥，唯一标识此会话
//   - initiator: 发起者的用户名
//   - data: 需要签名的数据
//   - accountAddr: 签名所使用的以太坊账户地址
//   - participants: 参与签名的用户列表
//
// 返回：
//   - 如果创建失败则返回错误信息
func (s *SignStorage) CreateSession(sessionKey, initiator, data, address string, participants []string) error {
	if sessionKey == "" || initiator == "" || data == "" || address == "" || len(participants) == 0 {
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
		return ErrOperationFailed
	}
	if count > 0 {
		return ErrSessionExists
	}

	// 创建新会话
	session := model.SignSession{
		SessionKey:   sessionKey,
		Initiator:    initiator,
		Data:         data,
		Address:      address,
		Participants: model.StringSlice(participants),
		Responses:    makeWaitingResponses(participants),
		Status:       model.StatusCreated,
	}

	if err := database.Create(&session).Error; err != nil {
		log.Printf("创建签名会话失败: %v", err)
		return ErrOperationFailed
	}

	return nil
}

// GetSession 获取指定密钥ID的签名会话
// 参数：
//   - sessionKey: 会话密钥，用于查找特定会话
//
// 返回：
//   - 会话对象指针
//   - 如果会话不存在或查询失败则返回错误信息
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
		return nil, ErrOperationFailed
	}
	return &session, nil
}

// UpdateStatus 更新签名会话状态
// 参数：
//   - sessionKey: 会话密钥，用于定位会话
//   - status: 新的会话状态
//
// 返回：
//   - 如果更新失败则返回错误信息
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
		return ErrOperationFailed
	}

	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// UpdateSignature 更新最终签名结果并将状态标记为已完成
// 参数：
//   - sessionKey: 会话密钥，用于定位会话
//   - signature: 最终的签名结果
//
// 返回：
//   - 如果更新失败则返回错误信息
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
		return ErrOperationFailed
	}

	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// DeleteSession 删除指定密钥ID的签名会话
// 参数：
//   - sessionKey: 会话密钥，用于定位要删除的会话
//
// 返回：
//   - 如果删除失败则返回错误信息
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
		return ErrOperationFailed
	}

	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// 在文件开头添加以下函数

// 在UpdateSignature函数之前添加UpdateParticipantStatus函数

// UpdateParticipantStatus 更新指定会话中某个参与者的状态
// 参数：
//   - sessionKey: 会话密钥，用于定位会话
//   - index: 参与者在数组中的索引
//   - status: 新的状态（如 accepted、rejected、completed 等）
//
// 返回：
//   - 如果更新失败则返回错误信息
func (s *SignStorage) UpdateParticipantStatus(sessionKey string, index int, status model.ParticipantStatus) error {
	if sessionKey == "" || index < 0 || status == "" {
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
		return ErrOperationFailed
	}

	// 验证索引是否有效
	if index >= len(session.Participants) {
		log.Printf("更新失败: 索引 %d 超出参与者列表范围 (长度: %d)", index, len(session.Participants))
		return ErrInvalidParameter
	}

	// 确保 Responses 数组长度足够
	if len(session.Responses) <= index {
		// 将 Responses 扩展到与 Participants 相同长度
		newResponses := make(model.StringSlice, len(session.Participants))
		copy(newResponses, session.Responses)
		session.Responses = newResponses
	}

	// 更新状态
	session.Responses[index] = string(status)

	// 保存更新
	if err := database.Save(&session).Error; err != nil {
		log.Printf("更新签名会话参与者状态失败: %v", err)
		return ErrOperationFailed
	}

	return nil
}

// 在DeleteSession函数之前添加UpdateChips函数

// UpdateChips 更新指定会话的 Chips 字段
// 参数：
//   - sessionKey: 会话密钥，用于定位会话
//   - chips: 新的 Chips 数组
//
// 返回：
//   - 如果更新失败则返回错误信息
func (s *SignStorage) UpdateChips(sessionKey string, chips []string) error {
	if sessionKey == "" || len(chips) == 0 {
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
		return ErrOperationFailed
	}

	// 验证 Chips 数组长度是否与 Participants 数组长度匹配
	if len(chips) != len(session.Participants) {
		log.Printf("更新失败: Chips 数组长度 (%d) 与 Participants 数组长度 (%d) 不匹配", len(chips), len(session.Participants))
		return ErrInvalidParameter
	}

	// 更新 Chips 字段
	session.Chips = model.StringSlice(chips)

	// 保存更新
	if err := database.Save(&session).Error; err != nil {
		log.Printf("更新签名会话 Chips 失败: %v", err)
		return ErrOperationFailed
	}

	return nil
}

// 在文件末尾添加以下函数

// AllKeyGenInvitationsAccepted 检查所有参与者是否接受了邀请
func (s *SignStorage) AllKeyGenInvitationsAccepted(sessionKey string) bool {
	if sessionKey == "" {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return false
	}

	var session model.SignSession
	if err := database.Where("session_key = ?", sessionKey).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false
		}
		log.Printf("获取签名会话失败: %v", err)
		return false
	}

	for _, response := range session.Responses {
		if response != string(model.ParticipantAccepted) {
			return false // 只要有一个参与者未接受邀请，返回 false
		}
	}

	return true // 所有参与者都已接受邀请
}

// AllKeyGenPartsCompleted 检查所有参与者是否完成了签名
func (s *SignStorage) AllKeyGenPartsCompleted(sessionKey string) bool {
	if sessionKey == "" {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return false
	}

	var session model.SignSession
	if err := database.Where("session_key = ?", sessionKey).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false
		}
		log.Printf("获取签名会话失败: %v", err)
		return false
	}

	for _, response := range session.Responses {
		if response != string(model.ParticipantCompleted) {
			return false // 只要有一个参与者未完成，返回 false
		}
	}

	return true // 所有参与者都已完成
}
