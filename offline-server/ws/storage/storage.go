package storage

import (
	"fmt"
	"log"
	"sync"

	"offline-server/storage"
	"offline-server/storage/db"
	"offline-server/storage/model"
)

// SessionManager 提供对会话的内存缓存和操作
type SessionManager struct {
	mu          sync.RWMutex
	keyGenCache map[string]*model.KeyGenSession
	signCache   map[string]*model.SignSession
}

var (
	managerInstance *SessionManager
	managerOnce     sync.Once
)

// GetSessionManager 返回SessionManager的单例实例
func GetSessionManager() *SessionManager {
	managerOnce.Do(func() {
		managerInstance = &SessionManager{
			keyGenCache: make(map[string]*model.KeyGenSession),
			signCache:   make(map[string]*model.SignSession),
		}
	})
	return managerInstance
}

// CreateKeyGenSession 在内存中创建一个新的密钥生成会话（不立即保存到数据库）
func (m *SessionManager) CreateKeyGenSession(sessionKey, initiator string, threshold, totalParts int, participants []string) error {
	if sessionKey == "" || initiator == "" || threshold <= 0 || totalParts <= 0 || len(participants) == 0 || threshold > totalParts {
		return storage.ErrInvalidParameter
	}

	// 创建新会话
	session := &model.KeyGenSession{
		SessionKey:   sessionKey,
		Initiator:    initiator,
		Threshold:    threshold,
		TotalParts:   totalParts,
		Participants: model.StringSlice(participants),
		Responses:    makeWaitingResponses(participants),
		Chips:        makeDefaultChips(len(participants)),
		Status:       model.StatusCreated,
	}

	// 添加到缓存
	m.mu.Lock()
	m.keyGenCache[sessionKey] = session
	m.mu.Unlock()

	return nil
}

// CreateSignSession 在内存中创建一个新的签名会话（不立即保存到数据库）
func (m *SessionManager) CreateSignSession(sessionKey, initiator, data, address string, participants []string) (*model.SignSession, error) {
	if sessionKey == "" || initiator == "" || data == "" || address == "" || len(participants) == 0 {
		return nil, storage.ErrInvalidParameter
	}

	// 创建新会话
	session := &model.SignSession{
		SessionKey:   sessionKey,
		Initiator:    initiator,
		Data:         data,
		Address:      address,
		Participants: model.StringSlice(participants),
		Responses:    makeWaitingResponses(participants),
		Status:       model.StatusCreated,
	}

	// 添加到缓存
	m.mu.Lock()
	m.signCache[sessionKey] = session
	m.mu.Unlock()

	return session, nil
}

// GetKeyGenSession 获取密钥生成会话，如果不在内存则从数据库加载
func (m *SessionManager) GetKeyGenSession(sessionKey string) *model.KeyGenSession {
	if sessionKey == "" {
		log.Fatal("GetKeyGenSession: sessionKey 不能为空")
	}

	// 先检查缓存
	m.mu.RLock()
	session, exists := m.keyGenCache[sessionKey]
	m.mu.RUnlock()

	if !exists {
		log.Fatal("GetKeyGenSession: 会话不存在")
	}
	return session
}

// GetSignSession 获取签名会话，如果不在内存则从数据库加载
func (m *SessionManager) GetSignSession(sessionKey string) *model.SignSession {
	if sessionKey == "" {
		log.Fatal("GetSignSession: sessionKey 不能为空")
	}

	// 先检查缓存
	m.mu.RLock()
	session, exists := m.signCache[sessionKey]
	m.mu.RUnlock()

	if !exists {
		log.Fatal("GetSignSession: 会话不存在")
	}
	return session
}

// SaveKeyGenSession 保存单个密钥生成会话到数据库
func (m *SessionManager) SaveKeyGenSession(session *model.KeyGenSession) error {
	if session == nil {
		return storage.ErrInvalidParameter
	}

	database := db.GetDB()
	if database == nil {
		return storage.ErrDatabaseNotInitialized
	}

	// 检查会话是否存在
	var count int64
	if err := database.Model(&model.KeyGenSession{}).Where("session_key = ?", session.SessionKey).Count(&count).Error; err != nil {
		return storage.ErrOperationFailed
	}

	var err error
	if count > 0 {
		// 更新现有记录
		err = database.Save(session).Error
	} else {
		// 创建新记录
		err = database.Create(session).Error
	}

	if err != nil {
		return storage.ErrOperationFailed
	}

	return nil
}

// SaveSignSession 保存单个签名会话到数据库
func (m *SessionManager) SaveSignSession(session *model.SignSession) error {
	if session == nil {
		return storage.ErrInvalidParameter
	}

	database := db.GetDB()
	if database == nil {
		return storage.ErrDatabaseNotInitialized
	}

	// 检查会话是否存在
	var count int64
	if err := database.Model(&model.SignSession{}).Where("session_key = ?", session.SessionKey).Count(&count).Error; err != nil {
		return storage.ErrOperationFailed
	}

	var err error
	if count > 0 {
		// 更新现有记录
		err = database.Save(session).Error
	} else {
		// 创建新记录
		err = database.Create(session).Error
	}

	if err != nil {
		return storage.ErrOperationFailed
	}

	return nil
}

// SaveAllSessions 将所有内存中的会话保存到数据库
func (m *SessionManager) SaveAllSessions() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 保存所有KeyGenSessions
	for _, session := range m.keyGenCache {
		if err := m.SaveKeyGenSession(session); err != nil {
			return err
		}
	}

	// 保存所有SignSessions
	for _, session := range m.signCache {
		if err := m.SaveSignSession(session); err != nil {
			return err
		}
	}

	return nil
}

// makeWaitingResponses 创建一个与参与者列表等长的响应数组，所有用户初始状态为等待邀请响应
func makeWaitingResponses(participants []string) model.StringSlice {
	responses := make([]string, len(participants))
	for i := range responses {
		responses[i] = string(model.ParticipantInit)
	}
	return responses
}

// makeDefaultChips 创建一个与参与者列表等长的 Chips 数组，默认值从 "SE000" 开始递增
func makeDefaultChips(count int) model.StringSlice {
	chips := make([]string, count)
	for i := range chips {
		chips[i] = fmt.Sprintf("SE%03d", i)
	}
	return chips
}
