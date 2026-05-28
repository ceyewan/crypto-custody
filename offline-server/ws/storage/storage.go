package storage

import (
	"sync"

	"offline-server/storage"
	"offline-server/storage/db"
	"offline-server/storage/model"
)

// SessionManager 提供对会话的内存缓存和操作
type SessionManager struct {
	mu            sync.RWMutex
	keyGenCache   map[string]*model.KeyGenSession
	signCache     map[string]*model.SignSession
	destroyCache  map[string]*DestroySession
	transferCache map[string]*TransferSession
}

// DestroySession 是密钥销毁 WebSocket 流程的内存会话。
type DestroySession struct {
	SessionKey   string
	OfflineKeyID string
	Initiator    string
	Address      string
	Participants model.StringSlice
	Responses    model.StringSlice
	Shards       []model.KeyShard
	Status       model.SessionStatus
	Reason       string
}

// TransferSession 是分片移交 WebSocket 双确认会话。
type TransferSession struct {
	SessionKey   string
	ShardID      string
	OfflineKeyID string
	Initiator    string
	Address      string
	CaseNo       string
	ShardIndex   int
	FromUsername string
	ToUsername   string
	Participants model.StringSlice
	Responses    model.StringSlice
	Status       model.SessionStatus
	Reason       string
}

var (
	managerInstance *SessionManager
	managerOnce     sync.Once
)

// GetSessionManager 返回SessionManager的单例实例
func GetSessionManager() *SessionManager {
	managerOnce.Do(func() {
		managerInstance = NewSessionManager()
	})
	return managerInstance
}

// NewSessionManager 创建独立的内存会话管理器，主要用于测试或显式依赖注入。
func NewSessionManager() *SessionManager {
	return &SessionManager{
		keyGenCache:   make(map[string]*model.KeyGenSession),
		signCache:     make(map[string]*model.SignSession),
		destroyCache:  make(map[string]*DestroySession),
		transferCache: make(map[string]*TransferSession),
	}
}

// CreateKeyGenSession 在内存中创建一个新的密钥生成会话（不立即保存到数据库）
func (m *SessionManager) CreateKeyGenSession(session model.KeyGenSession) error {
	if session.SessionKey == "" || session.Initiator == "" || session.RequiredSigners <= 0 ||
		session.TotalParties <= 0 || len(session.Participants) == 0 ||
		session.RequiredSigners > session.TotalParties {
		return storage.ErrInvalidParameter
	}
	if session.GG20Threshold == 0 {
		session.GG20Threshold = session.RequiredSigners - 1
	}
	if len(session.Responses) == 0 {
		session.Responses = makeWaitingResponses(session.Participants)
	}
	if session.Status == "" {
		session.Status = model.StatusCreated
	}

	// 添加到缓存
	m.mu.Lock()
	m.keyGenCache[session.SessionKey] = &session
	m.mu.Unlock()

	return nil
}

// CreateSignSession 在内存中创建一个新的签名会话（不立即保存到数据库）
func (m *SessionManager) CreateSignSession(session model.SignSession) (*model.SignSession, error) {
	if session.SessionKey == "" || session.Initiator == "" || session.MessageHash == "" ||
		session.Address == "" || len(session.Participants) == 0 {
		return nil, storage.ErrInvalidParameter
	}
	if len(session.Responses) == 0 {
		session.Responses = makeWaitingResponses(session.Participants)
	}
	if session.Status == "" {
		session.Status = model.StatusCreated
	}

	// 添加到缓存
	m.mu.Lock()
	m.signCache[session.SessionKey] = &session
	m.mu.Unlock()

	return &session, nil
}

// CreateDestroySession 在内存中创建一个新的密钥销毁会话。
func (m *SessionManager) CreateDestroySession(session DestroySession) (*DestroySession, error) {
	if session.SessionKey == "" || session.OfflineKeyID == "" || session.Initiator == "" ||
		session.Address == "" || len(session.Participants) == 0 || len(session.Shards) == 0 ||
		len(session.Participants) != len(session.Shards) {
		return nil, storage.ErrInvalidParameter
	}
	if len(session.Responses) == 0 {
		session.Responses = makeWaitingResponses(session.Participants)
	}
	if session.Status == "" {
		session.Status = model.StatusCreated
	}

	m.mu.Lock()
	m.destroyCache[session.SessionKey] = &session
	m.mu.Unlock()

	return &session, nil
}

// CreateTransferSession 在内存中创建分片移交会话。
func (m *SessionManager) CreateTransferSession(session TransferSession) (*TransferSession, error) {
	if session.SessionKey == "" || session.ShardID == "" || session.Initiator == "" ||
		session.Address == "" || session.FromUsername == "" || session.ToUsername == "" ||
		len(session.Participants) != 2 {
		return nil, storage.ErrInvalidParameter
	}
	if len(session.Responses) == 0 {
		session.Responses = makeWaitingResponses(session.Participants)
	}
	if session.Status == "" {
		session.Status = model.StatusCreated
	}

	m.mu.Lock()
	m.transferCache[session.SessionKey] = &session
	m.mu.Unlock()

	return &session, nil
}

// GetKeyGenSession 获取密钥生成会话，如果不在内存则从数据库加载
func (m *SessionManager) GetKeyGenSession(sessionKey string) *model.KeyGenSession {
	if sessionKey == "" {
		return nil
	}

	// 先检查缓存
	m.mu.RLock()
	session, exists := m.keyGenCache[sessionKey]
	m.mu.RUnlock()

	if !exists {
		return nil
	}
	return session
}

// GetSignSession 获取签名会话，如果不在内存则从数据库加载
func (m *SessionManager) GetSignSession(sessionKey string) *model.SignSession {
	if sessionKey == "" {
		return nil
	}

	// 先检查缓存
	m.mu.RLock()
	session, exists := m.signCache[sessionKey]
	m.mu.RUnlock()

	if !exists {
		return nil
	}
	return session
}

// GetDestroySession 获取密钥销毁会话。
func (m *SessionManager) GetDestroySession(sessionKey string) *DestroySession {
	if sessionKey == "" {
		return nil
	}

	m.mu.RLock()
	session, exists := m.destroyCache[sessionKey]
	m.mu.RUnlock()

	if !exists {
		return nil
	}
	return session
}

// GetTransferSession 获取分片移交会话。
func (m *SessionManager) GetTransferSession(sessionKey string) *TransferSession {
	if sessionKey == "" {
		return nil
	}

	m.mu.RLock()
	session, exists := m.transferCache[sessionKey]
	m.mu.RUnlock()
	if !exists {
		return nil
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
