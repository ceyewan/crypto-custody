// Package storage 提供对系统持久化数据的存储和访问管理
package storage

import (
	"errors"
)

// 系统中常见的错误定义
var (
	// ErrSessionExists 表示当尝试创建已存在的会话时返回的错误
	// 当使用相同的会话ID创建新会话时会触发此错误
	ErrSessionExists = errors.New("会话已存在")

	// ErrSessionNotFound 表示当请求的会话不存在时返回的错误
	// 在尝试获取、更新或删除不存在的会话时会触发此错误
	ErrSessionNotFound = errors.New("会话不存在")

	// ErrDatabaseNotInitialized 表示当数据库未初始化时返回的错误
	// 在尝试进行数据库操作但数据库连接尚未建立时会触发此错误
	ErrDatabaseNotInitialized = errors.New("数据库未初始化")

	// ErrInvalidParameter 表示当提供的参数无效时返回的错误
	// 当参数为空、格式错误或超出有效范围时会触发此错误
	ErrInvalidParameter = errors.New("参数无效")

	// ErrUserExists 表示当尝试创建已存在的用户时返回的错误
	// 在使用已存在的用户名或邮箱创建新用户时会触发此错误
	ErrUserExists = errors.New("用户已存在")

	// ErrUserNotFound 表示当请求的用户不存在时返回的错误
	// 在尝试获取、更新或删除不存在的用户时会触发此错误
	ErrUserNotFound = errors.New("用户不存在")

	// ErrInvalidCredentials 表示当用户提供的凭证无效时返回的错误
	// 在用户登录时提供错误的用户名或密码会触发此错误
	ErrInvalidCredentials = errors.New("用户名或密码错误")

	// ErrInvalidRole 表示当提供的角色类型无效时返回的错误
	// 在尝试将用户角色设置为非预定义角色时会触发此错误
	ErrInvalidRole = errors.New("无效的角色类型")

	// ErrRecordNotFound 表示当请求的记录不存在时返回的错误
	// 在尝试获取、更新或删除不存在的数据记录时会触发此错误
	ErrRecordNotFound = errors.New("记录未找到")

	// ErrParticipantNotFound 表示当请求的参与者不存在时返回的错误
	// 在会话中尝试操作不存在的参与者时会触发此错误
	ErrParticipantNotFound = errors.New("参与者未找到")

	// ErrSeExists 表示当尝试创建已存在的安全芯片记录时返回的错误
	// 在使用已存在的SeId或CPIC创建新记录时会触发此错误
	ErrSeExists = errors.New("安全芯片记录已存在")

	// ErrOperationFailed 表示当操作失败时返回的通用错误
	// 在数据库操作或其他系统操作失败时会触发此错误
	ErrOperationFailed = errors.New("操作失败")
)
