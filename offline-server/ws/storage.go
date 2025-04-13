package ws

import (
	"fmt"
	"sync"
)

// Storage 定义了状态存储接口
// 提供客户端连接管理、密钥生成会话管理、签名会话管理以及用户密钥分享管理等功能
type Storage interface {
	// AddClient 添加客户端连接
	// userID 为客户端唯一标识，conn 为客户端连接对象
	AddClient(userID string, conn interface{})

	// RemoveClient 移除客户端连接
	// userID 为要移除的客户端标识
	RemoveClient(userID string)

	// GetClient 获取客户端连接
	// 返回客户端连接对象和是否存在的标志
	GetClient(userID string) (interface{}, bool)

	// GetAllClients 获取所有客户端连接
	// 返回所有客户端连接的映射表
	GetAllClients() map[string]interface{}

	// SetClientRole 设置客户端角色
	// userID 为客户端标识，role 为角色名称
	SetClientRole(userID, role string)

	// GetClientRole 获取客户端角色
	// 返回客户端角色和是否存在的标志
	GetClientRole(userID string) (string, bool)

	// CreateKeyGenSession 创建密钥生成会话
	// keyID 为密钥标识，threshold 为阈值，participants 为参与者列表
	CreateKeyGenSession(keyID string, threshold int, participants []string) error

	// GetKeyGenSession 获取密钥生成会话
	// 返回会话对象和是否存在的标志
	GetKeyGenSession(keyID string) (*KeyGenSession, bool)

	// UpdateKeyGenSession 更新密钥生成会话
	// 通过 updateFn 函数更新会话状态
	UpdateKeyGenSession(keyID string, updateFn func(*KeyGenSession)) error

	// DeleteKeyGenSession 删除密钥生成会话
	DeleteKeyGenSession(keyID string)

	// CreateSignSession 创建签名会话
	// keyID 为密钥标识，data 为待签名数据，participants 为参与者列表
	CreateSignSession(keyID, data string, participants []string) error

	// GetSignSession 获取签名会话
	// 返回会话对象和是否存在的标志
	GetSignSession(keyID string) (*SignSession, bool)

	// UpdateSignSession 更新签名会话
	// 通过 updateFn 函数更新会话状态
	UpdateSignSession(keyID string, updateFn func(*SignSession)) error

	// DeleteSignSession 删除签名会话
	DeleteSignSession(keyID string)

	// AddUserShare 添加用户密钥分享
	// userID 为用户标识，keyID 为密钥标识，shareJSON 为分享的 JSON 字符串
	AddUserShare(userID, keyID, shareJSON string)

	// GetUserShare 获取用户密钥分享
	// 返回分享的 JSON 字符串和是否存在的标志
	GetUserShare(userID, keyID string) (string, bool)

	// GetUserShares 获取用户所有密钥分享
	// 返回用户所有密钥分享的映射表
	GetUserShares(userID string) map[string]string

	// RemoveUserShare 移除用户密钥分享
	RemoveUserShare(userID, keyID string)
}

// KeyGenSession 表示密钥生成会话
type KeyGenSession struct {
	KeyID        string          // 密钥标识
	Threshold    int             // 阈值，需要多少参与者才能完成操作
	Participants []string        // 参与者列表
	Responses    map[string]bool // 各参与者是否已响应
	Completed    map[string]bool // 各参与者是否已完成
}

// SignSession 表示签名会话
type SignSession struct {
	KeyID        string            // 密钥标识
	Data         string            // 要签名的数据
	Participants []string          // 参与者列表
	Responses    map[string]bool   // 各参与者是否已响应
	Results      map[string]string // 各参与者的签名结果
}

// MemoryStorage 是 Storage 接口的内存实现
// 提供线程安全的状态存储
type MemoryStorage struct {
	clients     map[string]interface{} // 客户端连接表
	clientRoles map[string]string      // 客户端角色表
	clientsLock sync.RWMutex           // 客户端数据锁

	keyGenStatus     map[string]*KeyGenSession // 密钥生成会话表
	keyGenStatusLock sync.RWMutex              // 密钥生成会话锁

	signStatus     map[string]*SignSession // 签名会话表
	signStatusLock sync.RWMutex            // 签名会话锁

	userShares     map[string]map[string]string // 用户密钥分享表 userID -> keyID -> shareJSON
	userSharesLock sync.RWMutex                 // 用户密钥分享锁
}

// NewMemoryStorage 创建并初始化一个新的内存存储实例
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		clients:      make(map[string]interface{}),
		clientRoles:  make(map[string]string),
		keyGenStatus: make(map[string]*KeyGenSession),
		signStatus:   make(map[string]*SignSession),
		userShares:   make(map[string]map[string]string),
	}
}

// AddClient 添加客户端连接
func (s *MemoryStorage) AddClient(userID string, conn interface{}) {
	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()
	s.clients[userID] = conn
}

// RemoveClient 移除客户端连接和角色
func (s *MemoryStorage) RemoveClient(userID string) {
	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()
	delete(s.clients, userID)
	delete(s.clientRoles, userID)
}

// GetClient 获取客户端连接
func (s *MemoryStorage) GetClient(userID string) (interface{}, bool) {
	s.clientsLock.RLock()
	defer s.clientsLock.RUnlock()
	conn, exists := s.clients[userID]
	return conn, exists
}

// GetAllClients 获取所有客户端连接的副本
func (s *MemoryStorage) GetAllClients() map[string]interface{} {
	s.clientsLock.RLock()
	defer s.clientsLock.RUnlock()
	copy := make(map[string]interface{})
	for k, v := range s.clients {
		copy[k] = v
	}
	return copy
}

// SetClientRole 设置客户端角色
func (s *MemoryStorage) SetClientRole(userID, role string) {
	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()
	s.clientRoles[userID] = role
}

// GetClientRole 获取客户端角色
func (s *MemoryStorage) GetClientRole(userID string) (string, bool) {
	s.clientsLock.RLock()
	defer s.clientsLock.RUnlock()
	role, exists := s.clientRoles[userID]
	return role, exists
}

// CreateKeyGenSession 创建密钥生成会话
// 如果会话已存在，将返回错误
func (s *MemoryStorage) CreateKeyGenSession(keyID string, threshold int, participants []string) error {
	s.keyGenStatusLock.Lock()
	defer s.keyGenStatusLock.Unlock()

	if _, exists := s.keyGenStatus[keyID]; exists {
		return fmt.Errorf("密钥生成会话已存在: %s", keyID)
	}

	s.keyGenStatus[keyID] = &KeyGenSession{
		KeyID:        keyID,
		Threshold:    threshold,
		Participants: participants,
		Responses:    make(map[string]bool),
		Completed:    make(map[string]bool),
	}
	return nil
}

// GetKeyGenSession 获取密钥生成会话
func (s *MemoryStorage) GetKeyGenSession(keyID string) (*KeyGenSession, bool) {
	s.keyGenStatusLock.RLock()
	defer s.keyGenStatusLock.RUnlock()
	session, exists := s.keyGenStatus[keyID]
	return session, exists
}

// UpdateKeyGenSession 更新密钥生成会话
// 如果会话不存在，将返回错误
func (s *MemoryStorage) UpdateKeyGenSession(keyID string, updateFn func(*KeyGenSession)) error {
	s.keyGenStatusLock.Lock()
	defer s.keyGenStatusLock.Unlock()
	session, exists := s.keyGenStatus[keyID]
	if !exists {
		return fmt.Errorf("密钥生成会话不存在: %s", keyID)
	}
	updateFn(session)
	return nil
}

// DeleteKeyGenSession 删除密钥生成会话
func (s *MemoryStorage) DeleteKeyGenSession(keyID string) {
	s.keyGenStatusLock.Lock()
	defer s.keyGenStatusLock.Unlock()
	delete(s.keyGenStatus, keyID)
}

// CreateSignSession 创建签名会话
// 如果会话已存在，将返回错误
func (s *MemoryStorage) CreateSignSession(keyID, data string, participants []string) error {
	s.signStatusLock.Lock()
	defer s.signStatusLock.Unlock()

	if _, exists := s.signStatus[keyID]; exists {
		return fmt.Errorf("签名会话已存在: %s", keyID)
	}

	s.signStatus[keyID] = &SignSession{
		KeyID:        keyID,
		Data:         data,
		Participants: participants,
		Responses:    make(map[string]bool),
		Results:      make(map[string]string),
	}
	return nil
}

// GetSignSession 获取签名会话
func (s *MemoryStorage) GetSignSession(keyID string) (*SignSession, bool) {
	s.signStatusLock.RLock()
	defer s.signStatusLock.RUnlock()
	session, exists := s.signStatus[keyID]
	return session, exists
}

// UpdateSignSession 更新签名会话
// 如果会话不存在，将返回错误
func (s *MemoryStorage) UpdateSignSession(keyID string, updateFn func(*SignSession)) error {
	s.signStatusLock.Lock()
	defer s.signStatusLock.Unlock()
	session, exists := s.signStatus[keyID]
	if !exists {
		return fmt.Errorf("签名会话不存在: %s", keyID)
	}
	updateFn(session)
	return nil
}

// DeleteSignSession 删除签名会话
func (s *MemoryStorage) DeleteSignSession(keyID string) {
	s.signStatusLock.Lock()
	defer s.signStatusLock.Unlock()
	delete(s.signStatus, keyID)
}

// AddUserShare 添加用户密钥分享
func (s *MemoryStorage) AddUserShare(userID, keyID, shareJSON string) {
	s.userSharesLock.Lock()
	defer s.userSharesLock.Unlock()
	if _, exists := s.userShares[userID]; !exists {
		s.userShares[userID] = make(map[string]string)
	}
	s.userShares[userID][keyID] = shareJSON
}

// GetUserShare 获取用户特定密钥的分享
func (s *MemoryStorage) GetUserShare(userID, keyID string) (string, bool) {
	s.userSharesLock.RLock()
	defer s.userSharesLock.RUnlock()
	if shares, exists := s.userShares[userID]; exists {
		share, exists := shares[keyID]
		return share, exists
	}
	return "", false
}

// GetUserShares 获取用户所有密钥分享的副本
func (s *MemoryStorage) GetUserShares(userID string) map[string]string {
	s.userSharesLock.RLock()
	defer s.userSharesLock.RUnlock()
	copy := make(map[string]string)
	if shares, exists := s.userShares[userID]; exists {
		for k, v := range shares {
			copy[k] = v
		}
	}
	return copy
}

// RemoveUserShare 移除用户特定密钥的分享
func (s *MemoryStorage) RemoveUserShare(userID, keyID string) {
	s.userSharesLock.Lock()
	defer s.userSharesLock.Unlock()
	if shares, exists := s.userShares[userID]; exists {
		delete(shares, keyID)
	}
}
