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
	UpdateUserRole(userID string, role string) error
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

	// UpdateResponse 更新参与者对会话的响应状态
	UpdateResponse(sessionKey, userName string, agreed bool) error

	// UpdateCompleted 更新参与者完成状态
	UpdateCompleted(sessionKey, userName string, completed bool) error

	// UpdateAccountAddr 更新会话关联的账户地址
	UpdateAccountAddr(sessionKey, accountAddr string) error

	// DeleteSession 删除指定密钥ID的生成会话
	DeleteSession(sessionKey string) error
}

// ISignStorage 定义签名会话存储接口
type ISignStorage interface {
	// CreateSession 创建新的签名会话
	CreateSession(sessionKey, initiator, data, accountAddr string, participants []string) error

	// GetSession 获取指定密钥ID的签名会话
	GetSession(sessionKey string) (*model.SignSession, error)

	// UpdateStatus 更新签名会话状态
	UpdateStatus(sessionKey string, status model.SessionStatus) error

	// UpdateResponse 更新参与者对会话的响应状态
	UpdateResponse(sessionKey, userName string, agreed bool) error

	// UpdateResult 更新参与者的签名结果
	UpdateResult(sessionKey, userName, result string) error

	// UpdateSignature 更新最终签名结果并将状态标记为已完成
	UpdateSignature(sessionKey, signature string) error

	// DeleteSession 删除指定密钥ID的签名会话
	DeleteSession(sessionKey string) error
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

// IKeyShardStorage 定义以太坊密钥分片存储接口
type IKeyShardStorage interface {
	// SaveKeyShard 保存以太坊密钥分片
	SaveKeyShard(username, address, pcic string, shardIndex int, privateShard string) (*model.EthereumKeyShard, error)

	// GetKeyShardByID 根据ID获取密钥分片
	GetKeyShardByID(id uint) (*model.EthereumKeyShard, error)

	// GetKeyShardByAddress 根据以太坊地址获取所有相关分片
	GetKeyShardByAddress(address string) ([]model.EthereumKeyShard, error)

	// GetKeyShardByUsername 根据用户名获取所有相关分片
	GetKeyShardByUsername(username string) ([]model.EthereumKeyShard, error)

	// GetKeyShardByAddressAndUsername 根据地址和用户名获取特定分片
	GetKeyShardByAddressAndUsername(address, username string) (*model.EthereumKeyShard, error)

	// GetKeyShardByAddressAndIndex 根据地址和分片索引获取特定分片
	GetKeyShardByAddressAndIndex(address string, index int) (*model.EthereumKeyShard, error)

	// UpdateKeyShard 更新密钥分片
	UpdateKeyShard(id uint, updates map[string]interface{}) error

	// DeleteKeyShard 删除密钥分片
	DeleteKeyShard(id uint) error

	// DeleteKeyShardByAddress 删除特定地址的所有分片
	DeleteKeyShardByAddress(address string) error
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
}
