package service

import (
	"errors"
	"offline-server/storage"
	"offline-server/storage/model"
	"strings"
)

// shareStorage 分享存储接口
var shareStorage storage.IShareStorage = storage.GetShareStorage()

// GetUserShare 获取用户特定密钥的分享数据
func GetUserShare(userName, sessionKey string) (map[string]interface{}, error) {
	// 参数验证
	if strings.TrimSpace(userName) == "" {
		return nil, errors.New("用户名不能为空")
	}
	if strings.TrimSpace(sessionKey) == "" {
		return nil, errors.New("会话密钥不能为空")
	}

	// 调用存储接口获取用户分享
	shareJSON, err := shareStorage.GetUserShare(userName, sessionKey)
	if err != nil {
		return nil, err
	}

	// 构建响应
	return map[string]interface{}{
		"key_id":     sessionKey,
		"user_name":  userName,
		"share_data": shareJSON,
	}, nil
}

// GetUserShares 获取用户所有分享数据
func GetUserShares(userName string) ([]map[string]interface{}, error) {
	// 参数验证
	if strings.TrimSpace(userName) == "" {
		return nil, errors.New("用户名不能为空")
	}

	// 调用存储接口获取用户所有分享
	sharesMap, err := shareStorage.GetUserShares(userName)
	if err != nil {
		return nil, err
	}

	// 构建响应
	shares := make([]map[string]interface{}, 0, len(sharesMap))
	for sessionKey, shareJSON := range sharesMap {
		shares = append(shares, map[string]interface{}{
			"key_id":     sessionKey,
			"user_name":  userName,
			"share_data": shareJSON,
		})
	}

	return shares, nil
}

// GetSessionShares 获取特定会话的所有分享数据（仅限管理员）
func GetSessionShares(sessionKey string) ([]map[string]interface{}, error) {
	// 参数验证
	if strings.TrimSpace(sessionKey) == "" {
		return nil, errors.New("会话密钥不能为空")
	}

	// 获取该会话相关的密钥生成会话信息
	session, err := keyGenStorage.GetSession(sessionKey)
	if err != nil {
		return nil, errors.New("获取会话信息失败")
	}

	// 收集所有参与者的分享
	shares := make([]map[string]interface{}, 0, len(session.Participants))
	for _, participant := range session.Participants {
		shareJSON, err := shareStorage.GetUserShare(participant, sessionKey)
		// 忽略未找到的分享
		if err == nil {
			shares = append(shares, map[string]interface{}{
				"key_id":     sessionKey,
				"user_name":  participant,
				"share_data": shareJSON,
			})
		}
	}

	return shares, nil
}

// GetKeyGenSessionByAccount 通过账户地址获取密钥生成会话
func GetKeyGenSessionByAccount(accountAddr string) (*model.KeyGenSession, error) {
	// 参数验证
	if strings.TrimSpace(accountAddr) == "" {
		return nil, errors.New("账户地址不能为空")
	}

	// 目前暂无直接按账户地址查询的接口，需要获取所有会话并筛选
	// 在实际应用中，应该增加一个按账户地址查询的存储接口方法

	// 返回错误，表示暂未实现此功能
	return nil, errors.New("功能暂未实现")
}

// GetSharesByAccount 通过账户地址获取所有相关分享
func GetSharesByAccount(accountAddr string) ([]map[string]interface{}, error) {
	// 参数验证
	if strings.TrimSpace(accountAddr) == "" {
		return nil, errors.New("账户地址不能为空")
	}

	// 获取与该账户相关的密钥生成会话
	session, err := GetKeyGenSessionByAccount(accountAddr)
	if err != nil {
		return nil, err
	}

	// 收集与会话相关的所有分享
	return GetSessionShares(session.SessionKey)
}
