package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"offline-server/storage"
	"offline-server/storage/model"
)

// signStorage 签名会话存储接口
var signStorage storage.ISignStorage = storage.GetSignStorage()

// CreateSignSession 创建签名会话
func CreateSignSession(initiator, keyID, data, accountAddr string, participants []string) (string, error) {
	// 参数验证
	if strings.TrimSpace(initiator) == "" {
		return "", errors.New("发起者不能为空")
	}
	if strings.TrimSpace(keyID) == "" {
		return "", errors.New("密钥ID不能为空")
	}
	if strings.TrimSpace(data) == "" {
		return "", errors.New("签名数据不能为空")
	}
	if strings.TrimSpace(accountAddr) == "" {
		return "", errors.New("账户地址不能为空")
	}
	if len(participants) == 0 {
		return "", errors.New("参与者列表不能为空")
	}

	// 验证密钥是否存在
	keyGenSession, err := keyGenStorage.GetSession(keyID)
	if err != nil {
		return "", fmt.Errorf("密钥不存在: %v", err)
	}

	// 验证账户地址是否匹配
	if keyGenSession.AccountAddr != accountAddr {
		return "", errors.New("账户地址与密钥不匹配")
	}

	// 验证参与者是否在密钥生成的参与者列表中
	validParticipants := make(map[string]bool)
	for _, p := range keyGenSession.Participants {
		validParticipants[p] = true
	}

	for _, p := range participants {
		if !validParticipants[p] {
			return "", fmt.Errorf("参与者 %s 不在密钥生成的参与者列表中", p)
		}
	}

	// 验证参与者数量是否满足阈值
	if len(participants) < keyGenSession.Threshold {
		return "", fmt.Errorf("参与者数量(%d)小于阈值(%d)", len(participants), keyGenSession.Threshold)
	}

	// 生成会话密钥
	timestamp := time.Now().Format("20060102150405")
	sessionKey := fmt.Sprintf("sign_%s_%s", timestamp, initiator)

	// 调用存储接口创建会话
	err = signStorage.CreateSession(
		sessionKey,
		initiator,
		data,
		accountAddr,
		participants,
	)
	if err != nil {
		return "", fmt.Errorf("创建签名会话失败: %v", err)
	}

	// 更新会话状态为已邀请
	err = signStorage.UpdateStatus(sessionKey, model.StatusInvited)
	if err != nil {
		return "", fmt.Errorf("更新会话状态失败: %v", err)
	}

	return sessionKey, nil
}

// GetSignSession 获取签名会话
func GetSignSession(sessionKey string) (*model.SignSession, error) {
	// 参数验证
	if strings.TrimSpace(sessionKey) == "" {
		return nil, errors.New("会话密钥不能为空")
	}

	// 调用存储接口获取会话
	session, err := signStorage.GetSession(sessionKey)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// UpdateSignStatus 更新签名会话状态
func UpdateSignStatus(sessionKey string, status model.SessionStatus) error {
	// 参数验证
	if strings.TrimSpace(sessionKey) == "" {
		return errors.New("会话密钥不能为空")
	}

	// 检查会话是否存在
	session, err := signStorage.GetSession(sessionKey)
	if err != nil {
		return err
	}

	// 验证状态转换是否有效
	if !isValidStatusTransition(session.Status, status) {
		return errors.New("无效的状态转换")
	}

	// 调用存储接口更新状态
	return signStorage.UpdateStatus(sessionKey, status)
}

// UpdateSignResponse 更新参与者对签名会话的响应
func UpdateSignResponse(sessionKey, userName string, agreed bool) error {
	// 参数验证
	if strings.TrimSpace(sessionKey) == "" {
		return errors.New("会话密钥不能为空")
	}
	if strings.TrimSpace(userName) == "" {
		return errors.New("用户名不能为空")
	}

	// 检查会话是否存在
	session, err := signStorage.GetSession(sessionKey)
	if err != nil {
		return err
	}

	// 检查用户是否是参与者
	isParticipant := false
	for _, participant := range session.Participants {
		if participant == userName {
			isParticipant = true
			break
		}
	}

	if !isParticipant {
		return errors.New("用户不是该会话的参与者")
	}

	// 调用存储接口更新响应
	return signStorage.UpdateResponse(sessionKey, userName, agreed)
}

// UpdateSignResult 更新参与者的签名结果
func UpdateSignResult(sessionKey, userName, result string) error {
	// 参数验证
	if strings.TrimSpace(sessionKey) == "" {
		return errors.New("会话密钥不能为空")
	}
	if strings.TrimSpace(userName) == "" {
		return errors.New("用户名不能为空")
	}
	if strings.TrimSpace(result) == "" {
		return errors.New("签名结果不能为空")
	}

	// 检查会话是否存在
	session, err := signStorage.GetSession(sessionKey)
	if err != nil {
		return err
	}

	// 检查用户是否是参与者
	isParticipant := false
	for _, participant := range session.Participants {
		if participant == userName {
			isParticipant = true
			break
		}
	}

	if !isParticipant {
		return errors.New("用户不是该会话的参与者")
	}

	// 调用存储接口更新签名结果
	return signStorage.UpdateResult(sessionKey, userName, result)
}

// UpdateSignature 更新最终签名结果
func UpdateSignature(sessionKey, signature string) error {
	// 参数验证
	if strings.TrimSpace(sessionKey) == "" {
		return errors.New("会话密钥不能为空")
	}
	if strings.TrimSpace(signature) == "" {
		return errors.New("签名结果不能为空")
	}

	// 检查会话是否存在
	_, err := signStorage.GetSession(sessionKey)
	if err != nil {
		return err
	}

	// 调用存储接口更新最终签名结果
	return signStorage.UpdateSignature(sessionKey, signature)
}

// GetSignSessionsByAccount 根据账户地址获取签名会话列表
func GetSignSessionsByAccount(accountAddr string) ([]*model.SignSession, error) {
	// 参数验证
	if strings.TrimSpace(accountAddr) == "" {
		return nil, errors.New("账户地址不能为空")
	}

	// 目前暂无直接按账户地址查询的接口，需要获取所有会话并筛选
	// 在实际应用中，应该增加一个按账户地址查询的存储接口方法
	// 这里仅作示例，实际应用中应该优化实现
	// 返回空数组，表示暂未实现此功能
	return []*model.SignSession{}, nil
}

// GetParticipantsByAccount 获取账户参与者信息
func GetParticipantsByAccount(accountAddr string) (*model.KeyGenSession, error) {
	// 参数验证
	if strings.TrimSpace(accountAddr) == "" {
		return nil, errors.New("账户地址不能为空")
	}

	session, err := keyGenStorage.GetSessionByAccountAddr(accountAddr)

	return session, err
}
