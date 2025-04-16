package ws

import (
	"fmt"
	"log"
	"sync"

	"gorm.io/gorm"

	"offline-server/storage"
	"offline-server/storage/db"
	"offline-server/storage/model"
)

// Storage 定义了状态存储接口
// 提供客户端连接管理、密钥生成会话管理、签名会话管理以及用户密钥分享管理等功能
type Storage interface {
	// 客户端连接管理
	AddClient(userID string, conn interface{})
	RemoveClient(userID string)
	GetClient(userID string) (interface{}, bool)
	GetAllClients() map[string]interface{}
	SetClientRole(userID, role string)
	GetClientRole(userID string) (string, bool)

	// 密钥生成会话管理
	CreateKeyGenSession(keyID string, threshold int, participants []string) error
	GetKeyGenSession(keyID string) (*KeyGenSession, bool)
	UpdateKeyGenSession(keyID string, updateFn func(*KeyGenSession)) error
	DeleteKeyGenSession(keyID string)

	// 签名会话管理
	CreateSignSession(keyID, data string, participants []string) error
	GetSignSession(keyID string) (*SignSession, bool)
	UpdateSignSession(keyID string, updateFn func(*SignSession)) error
	DeleteSignSession(keyID string)

	// 用户密钥分享管理
	AddUserShare(userID, keyID, shareJSON string)
	GetUserShare(userID, keyID string) (string, bool)
	GetUserShares(userID string) map[string]string
	RemoveUserShare(userID, keyID string)
}

// KeyGenSession 表示密钥生成会话
type KeyGenSession struct {
	KeyID        string          // 密钥标识
	Threshold    int             // 阈值，需要多少参与者才能完成操作
	Participants []string        // 参与者列表
	Responses    map[string]bool // 各参与者是否已响应
	Completed    map[string]bool // 各参与者是否已完成
	AccountAddr  string          // 账户地址
}

// SignSession 表示签名会话
type SignSession struct {
	KeyID        string            // 密钥标识
	Data         string            // 要签名的数据
	Participants []string          // 参与者列表
	Responses    map[string]bool   // 各参与者是否已响应
	Results      map[string]string // 各参与者的签名结果
	AccountAddr  string            // 账户地址
}

// PersistentStorage 是 Storage 接口的持久化实现
// 使用数据库存储会话信息，使用内存存储客户端连接
type PersistentStorage struct {
	clients     map[string]interface{} // 客户端连接表
	clientRoles map[string]string      // 客户端角色表
	clientsLock sync.RWMutex           // 客户端数据锁

	// 分离的存储实例
	shareStorage  storage.IShareStorage  // 用户分享存储
	keyGenStorage storage.IKeyGenStorage // 密钥生成会话存储
	signStorage   storage.ISignStorage   // 签名会话存储
}

// NewPersistentStorage 创建并初始化一个新的持久化存储实例
func NewPersistentStorage() *PersistentStorage {
	// 确保数据库已初始化
	if db.GetDB() == nil {
		if err := db.Init(); err != nil {
			log.Fatalf("初始化数据库失败: %v", err)
		}
	}

	// 自动迁移模型
	if err := db.AutoMigrate(&model.KeyGenSession{}, &model.SignSession{}, &model.UserShare{}); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	return &PersistentStorage{
		clients:       make(map[string]interface{}),
		clientRoles:   make(map[string]string),
		shareStorage:  storage.GetShareStorage(),
		keyGenStorage: storage.GetKeyGenStorage(),
		signStorage:   storage.GetSignStorage(),
	}
}

// AddClient 添加客户端连接
func (s *PersistentStorage) AddClient(userID string, conn interface{}) {
	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()
	s.clients[userID] = conn
}

// RemoveClient 移除客户端连接和角色
func (s *PersistentStorage) RemoveClient(userID string) {
	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()
	delete(s.clients, userID)
	delete(s.clientRoles, userID)
}

// GetClient 获取客户端连接
func (s *PersistentStorage) GetClient(userID string) (interface{}, bool) {
	s.clientsLock.RLock()
	defer s.clientsLock.RUnlock()
	conn, exists := s.clients[userID]
	return conn, exists
}

// GetAllClients 获取所有客户端连接的副本
func (s *PersistentStorage) GetAllClients() map[string]interface{} {
	s.clientsLock.RLock()
	defer s.clientsLock.RUnlock()
	copy := make(map[string]interface{})
	for k, v := range s.clients {
		copy[k] = v
	}
	return copy
}

// SetClientRole 设置客户端角色
func (s *PersistentStorage) SetClientRole(userID, role string) {
	s.clientsLock.Lock()
	defer s.clientsLock.Unlock()
	s.clientRoles[userID] = role
}

// GetClientRole 获取客户端角色
func (s *PersistentStorage) GetClientRole(userID string) (string, bool) {
	s.clientsLock.RLock()
	defer s.clientsLock.RUnlock()
	role, exists := s.clientRoles[userID]
	return role, exists
}

// CreateKeyGenSession 创建密钥生成会话
func (s *PersistentStorage) CreateKeyGenSession(keyID string, threshold int, participants []string) error {
	// 找出发起者（协调方）
	var initiatorID string
	for userID, role := range s.clientRoles {
		if role == "coordinator" {
			initiatorID = userID
			break
		}
	}

	// 创建持久化会话
	err := s.keyGenStorage.CreateSession(keyID, initiatorID, threshold, len(participants), participants)
	if err != nil {
		return err
	}

	// 更新状态
	return s.keyGenStorage.UpdateStatus(keyID, model.StatusCreated)
}

// GetKeyGenSession 获取密钥生成会话
func (s *PersistentStorage) GetKeyGenSession(keyID string) (*KeyGenSession, bool) {
	dbSession, err := s.keyGenStorage.GetSession(keyID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, false
		}
		log.Printf("获取密钥生成会话失败: %v", err)
		return nil, false
	}

	// 将数据库模型转换为内存模型
	session := &KeyGenSession{
		KeyID:        dbSession.KeyID,
		Threshold:    dbSession.Threshold,
		Participants: []string(dbSession.Participants),
		Responses:    map[string]bool(dbSession.Responses),
		Completed:    map[string]bool(dbSession.Completed),
		AccountAddr:  dbSession.AccountAddr,
	}

	return session, true
}

// UpdateKeyGenSession 更新密钥生成会话
func (s *PersistentStorage) UpdateKeyGenSession(keyID string, updateFn func(*KeyGenSession)) error {
	session, exists := s.GetKeyGenSession(keyID)
	if !exists {
		return fmt.Errorf("密钥生成会话不存在: %s", keyID)
	}

	// 应用更新
	updateFn(session)

	// 更新响应状态
	for userID, agreed := range session.Responses {
		if err := s.keyGenStorage.UpdateResponse(keyID, userID, agreed); err != nil {
			return err
		}
	}

	// 更新完成状态
	for userID, completed := range session.Completed {
		if err := s.keyGenStorage.UpdateCompleted(keyID, userID, completed); err != nil {
			return err
		}
	}

	// 更新账户地址
	if session.AccountAddr != "" {
		if err := s.keyGenStorage.UpdateAccountAddr(keyID, session.AccountAddr); err != nil {
			return err
		}
	}

	// 更新会话状态
	var status model.SessionStatus
	if len(session.Completed) == len(session.Participants) {
		allCompleted := true
		for _, completed := range session.Completed {
			if !completed {
				allCompleted = false
				break
			}
		}
		if allCompleted {
			status = model.StatusCompleted
		} else {
			status = model.StatusProcessing
		}
	} else if len(session.Responses) == len(session.Participants) {
		allAgreed := true
		for _, agreed := range session.Responses {
			if !agreed {
				allAgreed = false
				break
			}
		}
		if allAgreed {
			status = model.StatusAccepted
		} else {
			status = model.StatusRejected
		}
	} else if len(session.Responses) > 0 {
		status = model.StatusProcessing
	} else {
		status = model.StatusInvited
	}

	return s.keyGenStorage.UpdateStatus(keyID, status)
}

// DeleteKeyGenSession 删除密钥生成会话
func (s *PersistentStorage) DeleteKeyGenSession(keyID string) {
	_ = s.keyGenStorage.DeleteSession(keyID)
}

// CreateSignSession 创建签名会话
func (s *PersistentStorage) CreateSignSession(keyID, data string, participants []string) error {
	// 找出发起者（协调方）
	var initiatorID string
	for userID, role := range s.clientRoles {
		if role == "coordinator" {
			initiatorID = userID
			break
		}
	}

	// 获取账户地址（如果有）
	session, exists := s.GetKeyGenSession(keyID)
	var accountAddr string
	if exists {
		accountAddr = session.AccountAddr
	}

	// 创建持久化会话
	err := s.signStorage.CreateSession(keyID, initiatorID, data, accountAddr, participants)
	if err != nil {
		return err
	}

	// 更新状态
	return s.signStorage.UpdateStatus(keyID, model.StatusCreated)
}

// GetSignSession 获取签名会话
func (s *PersistentStorage) GetSignSession(keyID string) (*SignSession, bool) {
	dbSession, err := s.signStorage.GetSession(keyID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, false
		}
		log.Printf("获取签名会话失败: %v", err)
		return nil, false
	}

	// 将数据库模型转换为内存模型
	session := &SignSession{
		KeyID:        dbSession.KeyID,
		Data:         dbSession.Data,
		Participants: []string(dbSession.Participants),
		AccountAddr:  dbSession.AccountAddr,
	}

	// 确保 Responses 和 Results map 被初始化
	if dbSession.Responses != nil {
		session.Responses = map[string]bool(dbSession.Responses)
	} else {
		session.Responses = make(map[string]bool)
	}

	if dbSession.Results != nil {
		session.Results = map[string]string(dbSession.Results)
	} else {
		session.Results = make(map[string]string)
	}

	return session, true
}

// UpdateSignSession 更新签名会话
func (s *PersistentStorage) UpdateSignSession(keyID string, updateFn func(*SignSession)) error {
	session, exists := s.GetSignSession(keyID)
	if !exists {
		return fmt.Errorf("签名会话不存在: %s", keyID)
	}

	// 应用更新
	updateFn(session)

	// 更新响应状态
	for userID, agreed := range session.Responses {
		if err := s.signStorage.UpdateResponse(keyID, userID, agreed); err != nil {
			return err
		}
	}

	// 更新签名结果
	for userID, result := range session.Results {
		if err := s.signStorage.UpdateResult(keyID, userID, result); err != nil {
			return err
		}
	}

	// 更新会话状态
	var status model.SessionStatus
	if len(session.Results) == len(session.Participants) {
		// 如果所有参与者都提交了签名结果，则更新最终签名
		// 注意：实际情况可能需要先组合签名，然后再更新
		signature := ""
		for _, result := range session.Results {
			if signature == "" {
				signature = result
			}
		}
		if signature != "" {
			return s.signStorage.UpdateSignature(keyID, signature)
		}
		status = model.StatusCompleted
	} else if len(session.Responses) == len(session.Participants) {
		allAgreed := true
		for _, agreed := range session.Responses {
			if !agreed {
				allAgreed = false
				break
			}
		}
		if allAgreed {
			status = model.StatusAccepted
		} else {
			status = model.StatusRejected
		}
	} else if len(session.Responses) > 0 {
		status = model.StatusProcessing
	} else {
		status = model.StatusInvited
	}

	return s.signStorage.UpdateStatus(keyID, status)
}

// DeleteSignSession 删除签名会话
func (s *PersistentStorage) DeleteSignSession(keyID string) {
	_ = s.signStorage.DeleteSession(keyID)
}

// AddUserShare 添加用户密钥分享
func (s *PersistentStorage) AddUserShare(userID, keyID, shareJSON string) {
	_ = s.shareStorage.SaveUserShare(userID, keyID, shareJSON)
}

// GetUserShare 获取用户特定密钥的分享
func (s *PersistentStorage) GetUserShare(userID, keyID string) (string, bool) {
	shareJSON, err := s.shareStorage.GetUserShare(userID, keyID)
	if err != nil {
		return "", false
	}
	return shareJSON, true
}

// GetUserShares 获取用户所有密钥分享
func (s *PersistentStorage) GetUserShares(userID string) map[string]string {
	shares, err := s.shareStorage.GetUserShares(userID)
	if err != nil {
		return make(map[string]string)
	}
	return shares
}

// RemoveUserShare 移除用户密钥分享
func (s *PersistentStorage) RemoveUserShare(userID, keyID string) {
	_ = s.shareStorage.DeleteUserShare(userID, keyID)
}

// NewMemoryStorage 创建并初始化一个新的内存存储实例
// 此函数保留以兼容现有代码，但建议迁移到持久化存储
func NewMemoryStorage() Storage {
	log.Println("警告: 使用内存存储，数据将在服务重启后丢失")
	return NewPersistentStorage()
}
