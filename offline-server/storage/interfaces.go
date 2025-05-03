// Package storage 提供对系统持久化数据的存储和访问管理
package storage

import (
	"offline-server/storage/model"
)

// IUserStorage 定义用户账号存储接口
type IUserStorage interface {
	// CreateUser 创建新用户
	CreateUser(username, password, email string) (*model.User, error)

	// GetUserByCredentials 通过用户名和密码获取用户
	GetUserByCredentials(username, password string) (*model.User, error)

	// GetUserByUsername 根据用户名获取用户信息
	GetUserByUsername(username string) (*model.User, error)

	// GetAllUsers 获取所有用户列表
	GetAllUsers() ([]model.User, error)

	// UpdateUserRole 更新用户角色
	UpdateUserRole(username string, role string) error
}

// IShareStorage 定义以太坊私钥分片存储接口
type IShareStorage interface {
	// CreateEthereumKeyShard 创建以太坊私钥分片记录
	CreateEthereumKeyShard(username, address, pcic, privateShard string, shardIndex int) error

	// GetEthereumKeyShard 根据用户名和以太坊地址获取密钥分片数据
	GetEthereumKeyShard(username, address string) (*model.EthereumKeyShard, error)
}

// IKeyGenStorage 定义密钥生成会话存储接口
type IKeyGenStorage interface {
	// CreateSession 创建新的密钥生成会话
	CreateSession(sessionKey, initiator string, threshold, totalParts int, participants []string) error

	// GetSession 获取指定密钥ID的生成会话
	GetSession(sessionKey string) (*model.KeyGenSession, error)

	// GetSessionByAccountAddr 获取指定账户地址的生成会话
	GetSessionByAccountAddr(accountAddr string) (*model.KeyGenSession, error)

	// UpdateStatus 更新密钥生成会话状态
	UpdateStatus(sessionKey string, status model.SessionStatus) error

	// UpdateParticipantStatus 更新参与者的状态
	UpdateParticipantStatus(sessionKey string, index int, status model.ParticipantStatus) error

	// UpdateAccountAddr 更新会话关联的账户地址
	UpdateAccountAddr(sessionKey, accountAddr string) error

	// DeleteSession 删除指定密钥ID的生成会话
	DeleteSession(sessionKey string) error

	// UpdateChips 更新指定会话的 Chips 字段
	UpdateChips(sessionKey string, chips []string) error

	// AllKeyGenInvitationsAccepted 检查所有参与者是否接受了邀请
	AllKeyGenInvitationsAccepted(sessionKey string) bool

	// AllKeyGenPartsCompleted 检查所有参与者是否完成了密钥生成
	AllKeyGenPartsCompleted(sessionKey string) bool
}

// ISignStorage 定义签名会话存储接口
type ISignStorage interface {
	// CreateSession 创建新的签名会话
	CreateSession(sessionKey, initiator, data, accountAddr string, participants []string) error

	// GetSession 获取指定密钥ID的签名会话
	GetSession(sessionKey string) (*model.SignSession, error)

	// UpdateStatus 更新签名会话状态
	UpdateStatus(sessionKey string, status model.SessionStatus) error

	// UpdateParticipantStatus 更新参与者的状态
	UpdateParticipantStatus(sessionKey string, index int, status model.ParticipantStatus) error

	// UpdateSignature 更新最终签名结果并将状态标记为已完成
	UpdateSignature(sessionKey, signature string) error

	// DeleteSession 删除指定密钥ID的签名会话
	DeleteSession(sessionKey string) error

	// UpdateChips 更新指定会话的 Chips 字段
	UpdateChips(sessionKey string, chips []string) error

	// AllKeyGenInvitationsAccepted 检查所有参与者是否接受了邀请
	AllKeyGenInvitationsAccepted(sessionKey string) bool

	// AllKeyGenPartsCompleted 检查所有参与者是否完成了密钥生成
	AllKeyGenPartsCompleted(sessionKey string) bool
}

// ICaseStorage 定义案件存储接口
type ICaseStorage interface {
	// CreateCase 创建新案件
	CreateCase(name, description string, threshold, totalShards int) (*model.Case, error)

	// GetCaseByID 根据ID获取案件信息
	GetCaseByID(id uint) (*model.Case, error)

	// GetCaseByName 根据名称获取案件信息
	GetCaseByName(name string) (*model.Case, error)

	// GetCaseByAddress 根据以太坊地址获取案件信息
	GetCaseByAddress(address string) (*model.Case, error)

	// GetAllCases 获取所有案件列表
	GetAllCases() ([]model.Case, error)

	// UpdateCase 更新案件信息
	UpdateCase(id uint, updates map[string]interface{}) error

	// UpdateCaseStatus 更新案件状态
	UpdateCaseStatus(id uint, status model.CaseStatus) error

	// UpdateCaseAddress 更新案件关联的账户地址
	UpdateCaseAddress(id uint, address string) error

	// DeleteCase 删除案件
	DeleteCase(id uint) error
}

// ISeStorage 定义安全芯片存储接口
type ISeStorage interface {
	// CreateSe 创建新的安全芯片记录
	CreateSe(seId, cpic string) (*model.Se, error)

	// GetSeBySeId 根据安全芯片ID获取记录
	GetSeBySeId(seId string) (*model.Se, error)

	// GetSeByCPIC 根据CPIC获取安全芯片记录
	GetSeByCPIC(cpic string) (*model.Se, error)

	// GetAllSe 获取所有安全芯片记录
	GetAllSe() ([]model.Se, error)

	// GetRandomSeIds 随机选取指定数量的安全芯片ID
	GetRandomSeIds(n int) ([]string, error)
}
