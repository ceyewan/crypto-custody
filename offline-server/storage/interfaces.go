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

	// GetUserByID 根据ID获取用户信息
	GetUserByID(id uint) (*model.User, error)

	// GetAllUsers 获取所有用户列表
	GetAllUsers() ([]model.User, error)

	// UpdateUserRole 更新用户角色
	UpdateUserRole(userID uint, role string) error
}

// IShareStorage 定义用户密钥分享存储接口
type IShareStorage interface {
	// SaveUserShare 保存用户密钥分享
	SaveUserShare(userName, sessionKey, shareJSON string) error

	// GetUserShare 获取指定用户和密钥ID的分享数据
	GetUserShare(userName, sessionKey string) (string, error)

	// GetUserShares 获取指定用户的所有密钥分享数据
	GetUserShares(userName string) (map[string]string, error)

	// DeleteUserShare 删除指定用户和密钥ID的分享数据
	DeleteUserShare(userName, sessionKey string) error
}

// IKeyGenStorage 定义密钥生成会话存储接口
type IKeyGenStorage interface {
	// CreateSession 创建新的密钥生成会话
	CreateSession(sessionKey, initiator string, threshold, totalParts int, participants []string) error

	// GetSession 获取指定密钥ID的生成会话
	GetSession(sessionKey string) (*model.KeyGenSession, error)

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
