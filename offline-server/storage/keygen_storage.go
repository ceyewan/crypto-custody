// Package storage 提供对系统持久化数据的存储和访问管理
package storage

import (
	"fmt"
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
// 通过单例模式确保整个应用程序中只有一个存储实例
func GetKeyGenStorage() IKeyGenStorage {
	keyGenOnce.Do(func() {
		keyGenInstance = &KeyGenStorage{}
	})
	return keyGenInstance
}

// CreateSession 创建新的密钥生成会话
// 参数：
//   - sessionKey: 会话密钥，唯一标识此会话
//   - initiator: 发起者的用户名
//   - threshold: 密钥重建所需的最小分片数量
//   - totalParts: 密钥被拆分的总分片数量
//   - participants: 参与密钥生成的用户列表
//
// 返回：
//   - 如果创建失败则返回错误信息
func (s *KeyGenStorage) CreateSession(sessionKey, initiator string, threshold, totalParts int, participants []string) error {
	if sessionKey == "" || initiator == "" || threshold <= 0 || totalParts <= 0 || len(participants) == 0 || threshold > totalParts {
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
	if err := database.Model(&model.KeyGenSession{}).Where("session_key = ?", sessionKey).Count(&count).Error; err != nil {
		log.Printf("查询密钥生成会话失败: %v", err)
		return ErrOperationFailed
	}
	if count > 0 {
		return ErrSessionExists
	}

	// 创建新会话
	session := model.KeyGenSession{
		SessionKey:   sessionKey,
		Initiator:    initiator,
		Threshold:    threshold,
		TotalParts:   totalParts,
		Participants: model.StringSlice(participants),
		Responses:    makeWaitingResponses(participants),
		Chips:        makeDefaultChips(len(participants)),
		Status:       model.StatusCreated,
	}

	if err := database.Create(&session).Error; err != nil {
		log.Printf("创建密钥生成会话失败: %v", err)
		return ErrOperationFailed
	}

	return nil
}

// makeWaitingResponses 创建一个与参与者列表等长的响应数组，所有用户初始状态为等待邀请响应
func makeWaitingResponses(participants []string) model.StringSlice {
	responses := make(model.StringSlice, len(participants))
	for i := range responses {
		responses[i] = string(model.ParticipantInit) // 所有参与者初始状态为等待邀请响应
	}
	return responses
}

// makeDefaultChips 创建一个与参与者列表等长的 Chips 数组，默认值从 "SE000" 开始递增
func makeDefaultChips(count int) model.StringSlice {
	chips := make(model.StringSlice, count)
	for i := 0; i < count; i++ {
		chips[i] = fmt.Sprintf("SE%03d", i) // 格式化为 "SE000", "SE001", ...
	}
	return chips
}

// GetSession 获取指定密钥ID的生成会话
// 参数：
//   - sessionKey: 会话密钥，用于查找特定会话
//
// 返回：
//   - 会话对象指针
//   - 如果会话不存在或查询失败则返回错误信息
func (s *KeyGenStorage) GetSession(sessionKey string) (*model.KeyGenSession, error) {
	if sessionKey == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var session model.KeyGenSession
	if err := database.Where("session_key = ?", sessionKey).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrSessionNotFound
		}
		log.Printf("获取密钥生成会话失败: %v", err)
		return nil, ErrOperationFailed
	}
	return &session, nil
}

// GetSessionByAccountAddr 获取指定账户地址的密钥生成会话
// 参数：
//   - accountAddr: 以太坊账户地址
//
// 返回：
//   - 会话对象指针
//   - 如果会话不存在或查询失败则返回错误信息
func (s *KeyGenStorage) GetSessionByAccountAddr(accountAddr string) (*model.KeyGenSession, error) {
	if accountAddr == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var session model.KeyGenSession
	if err := database.Where("account_addr = ?", accountAddr).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrSessionNotFound
		}
		log.Printf("获取密钥生成会话失败: %v", err)
		return nil, ErrOperationFailed
	}
	return &session, nil
}

// UpdateStatus 更新密钥生成会话状态
// 参数：
//   - sessionKey: 会话密钥，用于定位会话
//   - status: 新的会话状态
//
// 返回：
//   - 如果更新失败则返回错误信息
func (s *KeyGenStorage) UpdateStatus(sessionKey string, status model.SessionStatus) error {
	if sessionKey == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	result := database.Model(&model.KeyGenSession{}).Where("session_key = ?", sessionKey).Update("status", status)
	if result.Error != nil {
		log.Printf("更新密钥生成会话状态失败: %v", result.Error)
		return ErrOperationFailed
	}

	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// UpdateParticipantStatus 更新指定会话中某个参与者的状态
// 参数：
//   - sessionKey: 会话密钥，用于定位会话
//   - index: 参与者在数组中的索引
//   - status: 新的状态（如 accepted、rejected、completed 等）
//
// 返回：
//   - 如果更新失败则返回错误信息
func (s *KeyGenStorage) UpdateParticipantStatus(sessionKey string, index int, status model.ParticipantStatus) error {
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
	var session model.KeyGenSession
	if err := database.Where("session_key = ?", sessionKey).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrSessionNotFound
		}
		log.Printf("获取密钥生成会话失败: %v", err)
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
		log.Printf("更新密钥生成会话参与者状态失败: %v", err)
		return ErrOperationFailed
	}

	return nil
}

// UpdateAccountAddr 更新会话关联的账户地址
// 参数：
//   - sessionKey: 会话密钥，用于定位会话
//   - accountAddr: 以太坊账户地址
//
// 返回：
//   - 如果更新失败则返回错误信息
func (s *KeyGenStorage) UpdateAccountAddr(sessionKey, accountAddr string) error {
	if sessionKey == "" || accountAddr == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	result := database.Model(&model.KeyGenSession{}).Where("session_key = ?", sessionKey).Update("account_addr", accountAddr)
	if result.Error != nil {
		log.Printf("更新密钥生成会话账户地址失败: %v", result.Error)
		return ErrOperationFailed
	}

	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// DeleteSession 删除指定密钥ID的生成会话
// 参数：
//   - sessionKey: 会话密钥，用于定位要删除的会话
//
// 返回：
//   - 如果删除失败则返回错误信息
func (s *KeyGenStorage) DeleteSession(sessionKey string) error {
	if sessionKey == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	result := database.Where("session_key = ?", sessionKey).Delete(&model.KeyGenSession{})
	if result.Error != nil {
		log.Printf("删除密钥生成会话失败: %v", result.Error)
		return ErrOperationFailed
	}

	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}

// UpdateChips 更新指定会话的 Chips 字段
// 参数：
//   - sessionKey: 会话密钥，用于定位会话
//   - chips: 新的 Chips 数组
//
// 返回：
//   - 如果更新失败则返回错误信息
func (s *KeyGenStorage) UpdateChips(sessionKey string, chips []string) error {
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
	var session model.KeyGenSession
	if err := database.Where("session_key = ?", sessionKey).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrSessionNotFound
		}
		log.Printf("获取密钥生成会话失败: %v", err)
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
		log.Printf("更新密钥生成会话 Chips 失败: %v", err)
		return ErrOperationFailed
	}

	return nil
}

// AllKeyGenInvitationsAccepted 检查所有参与者是否接受了邀请
func (s *KeyGenStorage) AllKeyGenInvitationsAccepted(sessionKey string) bool {
	if sessionKey == "" {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return false
	}

	var session model.KeyGenSession
	if err := database.Where("session_key = ?", sessionKey).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false
		}
		log.Printf("获取密钥生成会话失败: %v", err)
		return false
	}

	for _, response := range session.Responses {
		if response != string(model.ParticipantAccepted) {
			return false // 只要有一个参与者未接受邀请，返回 false
		}
	}

	return true // 所有参与者都已接受邀请
}

// AllKeyGenPartsCompleted 检查所有参与者是否完成了密钥生成
func (s *KeyGenStorage) AllKeyGenPartsCompleted(sessionKey string) bool {
	if sessionKey == "" {
		return false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return false
	}

	var session model.KeyGenSession
	if err := database.Where("session_key = ?", sessionKey).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false
		}
		log.Printf("获取密钥生成会话失败: %v", err)
		return false
	}

	for _, response := range session.Responses {
		if response != string(model.ParticipantCompleted) {
			return false // 只要有一个参与者未完成，返回 false
		}
	}

	return true // 所有参与者都已完成
}
