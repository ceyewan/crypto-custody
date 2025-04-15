// Package storage 提供对系统持久化数据的存储和访问管理
//
// 此文件已弃用，相关内容已重构并移至以下文件:
// - errors.go: 错误定义
// - interfaces.go: 存储接口定义
// - share_storage.go: 用户密钥分享存储实现
// - keygen_storage.go: 密钥生成会话存储实现
// - sign_storage.go: 签名会话存储实现
//
// 为了保持向后兼容性，此文件导入了必要的依赖，并提供重定向函数
package storage

// 为兼容旧代码，保留重定向函数，但实际使用新实现
// 在后续迭代中，应直接使用新实现的接口和函数

// GetShareStorageCompat 兼容旧代码的重定向函数，返回分享存储实例
// 推荐使用 GetShareStorage() 代替
func GetShareStorageCompat() *ShareStorage {
	return shareInstance
}

// GetKeyGenStorageCompat 兼容旧代码的重定向函数，返回密钥生成存储实例
// 推荐使用 GetKeyGenStorage() 代替
func GetKeyGenStorageCompat() *KeyGenStorage {
	return keyGenInstance
}

// GetSignStorageCompat 兼容旧代码的重定向函数，返回签名存储实例
// 推荐使用 GetSignStorage() 代替
func GetSignStorageCompat() *SignStorage {
	return signInstance
}
