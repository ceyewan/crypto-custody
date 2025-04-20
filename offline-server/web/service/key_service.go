package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"offline-server/storage"
	"offline-server/storage/model"
)

// keyGenStorage 密钥生成会话存储接口
var keyGenStorage storage.IKeyGenStorage = storage.GetKeyGenStorage()

// CreateKenGenSessionKey 创建密钥生成会话密钥
func CreateKenGenSessionKey(initiator string) (string, error) {
	// 参数验证
	if strings.TrimSpace(initiator) == "" {
		return "", errors.New("发起者不能为空")
	}

	// 生成会话密钥
	timestamp := time.Now().Format("20060102150405")
	sessionKey := fmt.Sprintf("genkey_%s_%s", timestamp, initiator)

	return sessionKey, nil
}

// CreateKeyGenSession 创建密钥生成会话
func CreateKeyGenSession(initiator string, threshold, total_parts int, participants []string) (string, error) {
	// 验证参数
	if threshold < 1 {
		return "", errors.New("阈值必须大于0")
	}
	if total_parts < threshold {
		return "", errors.New("参与者数量必须不少于阈值")
	}

	// 验证所有参与者用户名是否存在
	for _, participant := range participants {
		// 检查用户是否存在
		user, err := userStorage.GetUserByUsername(participant)
		if err != nil || user == nil {
			return "", fmt.Errorf("用户 %s 不存在", participant)
		}
	}

	// 生成会话密钥
	sessionKey, _ := CreateKenGenSessionKey(initiator)

	// 调用存储接口创建会话
	err := keyGenStorage.CreateSession(
		sessionKey,
		initiator,
		threshold,
		total_parts,
		participants,
	)
	if err != nil {
		return "", fmt.Errorf("创建密钥生成会话失败: %v", err)
	}

	// TODO: 发送邀请通知给参与者

	// 更新会话状态为已邀请
	err = keyGenStorage.UpdateStatus(sessionKey, model.StatusInvited)
	if err != nil {
		return "", fmt.Errorf("更新会话状态失败: %v", err)
	}

	return sessionKey, nil
}

// GetKeyGenSession 获取密钥生成会话
func GetKeyGenSession(sessionKey string) (*model.KeyGenSession, error) {
	// 参数验证
	if strings.TrimSpace(sessionKey) == "" {
		return nil, errors.New("会话密钥不能为空")
	}

	// 调用存储接口获取会话
	session, err := keyGenStorage.GetSession(sessionKey)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// UpdateKeyGenStatus 更新密钥生成会话状态
func UpdateKeyGenStatus(sessionKey string, status model.SessionStatus) error {
	// 参数验证
	if strings.TrimSpace(sessionKey) == "" {
		return errors.New("会话密钥不能为空")
	}

	// 检查会话是否存在
	session, err := keyGenStorage.GetSession(sessionKey)
	if err != nil {
		return err
	}

	// 验证状态转换是否有效
	if !isValidStatusTransition(session.Status, status) {
		return errors.New("无效的状态转换")
	}

	// 调用存储接口更新状态
	return keyGenStorage.UpdateStatus(sessionKey, status)
}

// UpdateKeyGenAccountAddr 更新密钥生成会话关联的账户地址
func UpdateKeyGenAccountAddr(sessionKey, accountAddr string) error {
	// 参数验证
	if strings.TrimSpace(sessionKey) == "" {
		return errors.New("会话密钥不能为空")
	}
	if strings.TrimSpace(accountAddr) == "" {
		return errors.New("账户地址不能为空")
	}

	// 调用存储接口更新账户地址
	return keyGenStorage.UpdateAccountAddr(sessionKey, accountAddr)
}

// UpdateKeyGenResponse 更新参与者对会话的响应
func UpdateKeyGenResponse(sessionKey, userName string, agreed bool) error {
	// 参数验证
	if strings.TrimSpace(sessionKey) == "" {
		return errors.New("会话密钥不能为空")
	}
	if strings.TrimSpace(userName) == "" {
		return errors.New("用户名不能为空")
	}

	// 检查会话是否存在
	session, err := keyGenStorage.GetSession(sessionKey)
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
	return keyGenStorage.UpdateResponse(sessionKey, userName, agreed)
}

// UpdateKeyGenCompleted 更新参与者完成状态
func UpdateKeyGenCompleted(sessionKey, userName string, completed bool) error {
	// 参数验证
	if strings.TrimSpace(sessionKey) == "" {
		return errors.New("会话密钥不能为空")
	}
	if strings.TrimSpace(userName) == "" {
		return errors.New("用户名不能为空")
	}

	// 检查会话是否存在
	session, err := keyGenStorage.GetSession(sessionKey)
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

	// 调用存储接口更新完成状态
	return keyGenStorage.UpdateCompleted(sessionKey, userName, completed)
}

// isValidStatusTransition 验证状态转换是否有效
func isValidStatusTransition(current, target model.SessionStatus) bool {
	// 定义状态转换规则
	validTransitions := map[model.SessionStatus][]model.SessionStatus{
		model.StatusCreated:                {model.StatusInvited},
		model.StatusInvited:                {model.StatusWaitingInviteResponse, model.StatusRejected},
		model.StatusWaitingInviteResponse:  {model.StatusAccepted, model.StatusRejected},
		model.StatusAccepted:               {model.StatusProcessing},
		model.StatusProcessing:             {model.StatusWaitingProcessResponse, model.StatusFailed},
		model.StatusWaitingProcessResponse: {model.StatusCompleted, model.StatusFailed},
	}

	// 检查转换是否有效
	transitions, exists := validTransitions[current]
	if !exists {
		return false
	}

	for _, validTransition := range transitions {
		if validTransition == target {
			return true
		}
	}

	return false
}
