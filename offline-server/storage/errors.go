// Package storage 提供对系统持久化数据的存储和访问管理
package storage

import (
	"errors"
)

// 定义常见错误
var (
	// ErrSessionExists 当尝试创建已存在的会话时返回
	ErrSessionExists = errors.New("会话已存在")

	// ErrSessionNotFound 当请求的会话不存在时返回
	ErrSessionNotFound = errors.New("会话不存在")

	// ErrDatabaseNotInitialized 当数据库未初始化时返回
	ErrDatabaseNotInitialized = errors.New("数据库未初始化")

	// ErrInvalidParameter 当参数无效时返回
	ErrInvalidParameter = errors.New("参数无效")
)
