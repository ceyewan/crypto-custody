// Package storage 提供对系统持久化数据的存储和访问管理
package storage

import (
	"offline-server/storage/model"
)

// IShareStorage 定义用户密钥分享存储接口
type IShareStorage interface {
	// SaveUserShare 保存用户密钥分享
	SaveUserShare(userID, keyID, shareJSON string) error

	// GetUserShare 获取指定用户和密钥ID的分享数据
	GetUserShare(userID, keyID string) (string, error)

	// GetUserShares 获取指定用户的所有密钥分享数据
	GetUserShares(userID string) (map[string]string, error)

	// DeleteUserShare 删除指定用户和密钥ID的分享数据
	DeleteUserShare(userID, keyID string) error
}

// IKeyGenStorage 定义密钥生成会话存储接口
type IKeyGenStorage interface {
	// CreateSession 创建新的密钥生成会话
	CreateSession(keyID, initiatorID string, threshold, totalParts int, participants []string) error

	// GetSession 获取指定密钥ID的生成会话
	GetSession(keyID string) (*model.KeyGenSession, error)

	// UpdateStatus 更新密钥生成会话状态
	UpdateStatus(keyID string, status model.SessionStatus) error

	// UpdateResponse 更新参与者对会话的响应状态
	UpdateResponse(keyID, userID string, agreed bool) error

	// UpdateCompleted 更新参与者完成状态
	UpdateCompleted(keyID, userID string, completed bool) error

	// UpdateAccountAddr 更新会话关联的账户地址
	UpdateAccountAddr(keyID, accountAddr string) error

	// DeleteSession 删除指定密钥ID的生成会话
	DeleteSession(keyID string) error
}

// ISignStorage 定义签名会话存储接口
type ISignStorage interface {
	// CreateSession 创建新的签名会话
	CreateSession(keyID, initiatorID, data, accountAddr string, participants []string) error

	// GetSession 获取指定密钥ID的签名会话
	GetSession(keyID string) (*model.SignSession, error)

	// UpdateStatus 更新签名会话状态
	UpdateStatus(keyID string, status model.SessionStatus) error

	// UpdateResponse 更新参与者对会话的响应状态
	UpdateResponse(keyID, userID string, agreed bool) error

	// UpdateResult 更新参与者的签名结果
	UpdateResult(keyID, userID, result string) error

	// UpdateSignature 更新最终签名结果并将状态标记为已完成
	UpdateSignature(keyID, signature string) error

	// DeleteSession 删除指定密钥ID的签名会话
	DeleteSession(keyID string) error
}
