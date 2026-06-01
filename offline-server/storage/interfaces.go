// Package storage 提供对系统持久化数据的存储和访问管理
package storage

import (
	"time"

	"offline-server/storage/model"
)

// AuditLogFilter 定义审计日志的服务端筛选条件。
type AuditLogFilter struct {
	Limit    int
	TimeFrom time.Time
	TimeTo   time.Time
	Username string
	Role     string
	Action   string
	Resource string
	CaseNo   string
	Address  string
	Result   string
}

// IUserStorage 定义用户账号存储接口
type IUserStorage interface {
	// CreateUser 创建新用户
	CreateUser(username, password, nickname string) (*model.User, error)

	// GetUserByCredentials 通过用户名和密码获取用户
	GetUserByCredentials(username, password string) (*model.User, error)

	// GetUserByUsername 根据用户名获取用户信息
	GetUserByUsername(username string) (*model.User, error)

	// GetAllUsers 获取所有用户列表
	GetAllUsers() ([]model.User, error)

	// UpdateUserRole 更新用户角色
	UpdateUserRole(username string, role string) error
}

// IShareStorage 定义离线密钥分片存储接口
type IShareStorage interface {
	// CreateKeyShard 创建密钥分片记录
	CreateKeyShard(shard model.KeyShard) (*model.KeyShard, error)

	// GetKeyShardForParticipant 根据用户名和地址获取可用分片
	GetKeyShardForParticipant(username, address string) (*model.KeyShard, error)

	// GetKeyShardByID 根据分片编号获取分片
	GetKeyShardByID(shardID string) (*model.KeyShard, error)

	// ListActiveKeyShardsByAddress 获取地址下所有可用分片
	ListActiveKeyShardsByAddress(address string) ([]model.KeyShard, error)

	// ListKeyShardsByAddress 获取地址下全部分片
	ListKeyShardsByAddress(address string) ([]model.KeyShard, error)

	// ListKeyShardsByUsername 获取某个用户持有的全部分片
	ListKeyShardsByUsername(username string) ([]model.KeyShard, error)

	// ListKeyShards 获取全部分片
	ListKeyShards() ([]model.KeyShard, error)

	// UpdateKeyShardStatus 更新分片状态
	UpdateKeyShardStatus(shardID string, status model.KeyShardStatus) error

	// TransferKeyShard 调整单个分片持有人，不改变 SE CPLC、record_id 或密文
	TransferKeyShard(shardID, newUsername string) (*model.KeyShard, error)
}

// IOfflineKeyStorage 定义离线密钥元数据存储接口
type IOfflineKeyStorage interface {
	CreateOfflineKey(key model.OfflineKey) (*model.OfflineKey, error)
	GetOfflineKeyByID(offlineKeyID string) (*model.OfflineKey, error)
	GetOfflineKeyByAddress(address string) (*model.OfflineKey, error)
	GetOfflineKeyByTaskNo(taskNo string) (*model.OfflineKey, error)
	ListOfflineKeys() ([]model.OfflineKey, error)
	UpdateOfflineKeyOwner(offlineKeyID, logicalOwner string) error
	UpdateOfflineKeyStatus(offlineKeyID string, status model.OfflineKeyStatus) error
}

// IKeyGenStorage 定义密钥生成会话存储接口
type IKeyGenStorage interface {
	// CreateSession 创建新的密钥生成会话
	CreateSession(session model.KeyGenSession) (*model.KeyGenSession, error)

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

	// UpdateSeIDs 更新指定会话的 SeIDs 字段
	UpdateSeIDs(sessionKey string, seIDs []string) error

	// AllKeyGenInvitationsAccepted 检查所有参与者是否接受了邀请
	AllKeyGenInvitationsAccepted(sessionKey string) bool

	// AllKeyGenPartsCompleted 检查所有参与者是否完成了密钥生成
	AllKeyGenPartsCompleted(sessionKey string) bool
}

// ISignStorage 定义签名会话存储接口
type ISignStorage interface {
	// CreateSession 创建新的签名会话
	CreateSession(session model.SignSession) (*model.SignSession, error)

	// GetSession 获取指定密钥ID的签名会话
	GetSession(sessionKey string) (*model.SignSession, error)

	// GetSessionByTaskNo 获取指定任务编号的签名会话
	GetSessionByTaskNo(taskNo string) (*model.SignSession, error)

	// UpdateStatus 更新签名会话状态
	UpdateStatus(sessionKey string, status model.SessionStatus) error

	// UpdateParticipantStatus 更新参与者的状态
	UpdateParticipantStatus(sessionKey string, index int, status model.ParticipantStatus) error

	// UpdateSignature 更新最终签名结果并将状态标记为已完成
	UpdateSignature(sessionKey, signature string) error

	// DeleteSession 删除指定密钥ID的签名会话
	DeleteSession(sessionKey string) error

	// UpdateSeIDs 更新指定会话的 SeIDs 字段
	UpdateSeIDs(sessionKey string, seIDs []string) error

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
	CreateSe(seID, cplc, custodyLocation, registeredBy string) (*model.Se, error)

	// GetSeBySeId 根据安全芯片ID获取记录
	GetSeBySeId(seId string) (*model.Se, error)

	// GetSeByCPLC 根据CPLC获取安全芯片记录
	GetSeByCPLC(cplc string) (*model.Se, error)

	// GetAllSe 获取所有安全芯片记录
	GetAllSe() ([]model.Se, error)

	// GetActiveSeIds 选取指定数量的可用安全芯片ID
	GetActiveSeIds(n int) ([]string, error)

	// UpdateSeStatus 更新安全芯片状态
	UpdateSeStatus(seID string, status model.SeStatus) error
}

// IOfflineTaskStorage 定义离线任务存储接口
type IOfflineTaskStorage interface {
	CreateTask(task model.OfflineTask) (*model.OfflineTask, error)
	GetTask(taskNo string) (*model.OfflineTask, error)
	ListTasks() ([]model.OfflineTask, error)
	UpdateTaskStatus(taskNo string, status model.OfflineTaskStatus) error
	UpdateTaskResultHash(taskNo, resultHash string) error
}

// IAuditStorage 定义审计日志存储接口
type IAuditStorage interface {
	CreateAuditLog(log model.AuditLog) error
	ListAuditLogs(limit int) ([]model.AuditLog, error)
	SearchAuditLogs(filter AuditLogFilter) ([]model.AuditLog, error)
}

// IApprovalStorage 定义敏感操作审批记录存储接口
type IApprovalStorage interface {
	CreateApproval(approval model.Approval) (*model.Approval, error)
	ListApprovals(limit int) ([]model.Approval, error)
}
