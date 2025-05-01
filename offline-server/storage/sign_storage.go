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
func (s *SignStorage) CreateSession(sessionKey, initiator, data, accountAddr string, participants []string) error {
	if sessionKey == "" || initiator == "" || data == "" || accountAddr == "" || len(participants) == 0 {
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
		AccountAddr:  accountAddr,
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

// UpdateResponse 更新参与者对会话的响应状态
// 参数：
//   - sessionKey: 会话密钥，用于定位会话
//   - userName: 参与者用户名
//   - agreed: 是否同意参与签名
//
// 返回：
//   - 如果更新失败则返回错误信息
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
		return ErrOperationFailed
	}

	// 查找参与者在数组中的索引
	participantIndex := -1
	for i, participant := range session.Participants {
		if participant == userName {
			participantIndex = i
			break
		}
	}

	// 如果找不到参与者，返回错误
	if participantIndex == -1 {
		log.Printf("参与者 %s 不在会话 %s 的参与列表中", userName, sessionKey)
		return ErrParticipantNotFound
	}

	// 确保 Responses 数组长度足够
	if len(session.Responses) <= participantIndex {
		// 将 Responses 扩展到与 Participants 相同长度
		newResponses := make(model.StringSlice, len(session.Participants))
		copy(newResponses, session.Responses)
		session.Responses = newResponses
	}

	// 更新响应状态
	status := model.StatusRejected
	if agreed {
		status = model.StatusAccepted
	}
	session.Responses[participantIndex] = string(status)

	// 保存更新
	if err := database.Save(&session).Error; err != nil {
		log.Printf("更新签名会话响应失败: %v", err)
		return ErrOperationFailed
	}

	return nil
}

// UpdateResult 更新参与者的签名结果
// 参数：
//   - sessionKey: 会话密钥，用于定位会话
//   - userName: 参与者用户名
//   - result: 参与者生成的部分签名结果
//
// 返回：
//   - 如果更新失败则返回错误信息
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
		return ErrOperationFailed
	}

	// 查找参与者在数组中的索引
	participantIndex := -1
	for i, participant := range session.Participants {
		if participant == userName {
			participantIndex = i
			break
		}
	}

	// 如果找不到参与者，返回错误
	if participantIndex == -1 {
		log.Printf("参与者 %s 不在会话 %s 的参与列表中", userName, sessionKey)
		return ErrParticipantNotFound
	}

	// 更新响应状态为已完成
	if len(session.Responses) <= participantIndex {
		// 将 Responses 扩展到与 Participants 相同长度
		newResponses := make(model.StringSlice, len(session.Participants))
		copy(newResponses, session.Responses)
		session.Responses = newResponses
	}
	session.Responses[participantIndex] = string(model.StatusCompleted)

	// 为会话添加部分签名结果字段
	updates := map[string]interface{}{
		"responses": session.Responses,
	}

	// 部分签名结果存储在signature字段
	// 每次只处理一个用户的结果，在这里我们简单地更新为最新结果
	// 在实际应用中，可能需要将多个部分签名结合起来
	updates["signature"] = result

	// 保存更新
	if err := database.Model(&model.SignSession{}).Where("session_key = ?", sessionKey).Updates(updates).Error; err != nil {
		log.Printf("更新参与者签名结果失败: %v", err)
		return ErrOperationFailed
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
