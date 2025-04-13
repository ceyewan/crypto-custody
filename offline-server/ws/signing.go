package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// HandleSignRequest 处理签名请求
// 解析请求载荷，创建签名会话，并向所有参与方发送签名邀请
//
// 参数:
//   - store: 存储接口，用于管理会话状态和客户端连接
//   - msg: 包含签名请求数据的消息
//
// 返回:
//   - error: 如果处理过程中出现错误，返回相应错误；否则返回nil
func HandleSignRequest(store Storage, msg Message) error {
	var payload SignRequestPayload
	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		log.Printf("序列化签名请求载荷失败: %v", err)
		return fmt.Errorf("序列化载荷失败: %w", err)
	}

	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		log.Printf("解析签名请求载荷失败: %v", err)
		return fmt.Errorf("解析载荷失败: %w", err)
	}

	keyID := payload.KeyID
	data := payload.Data
	participants := payload.Participants

	log.Printf("收到签名请求 KeyID: %s, Data: %s, 参与方: %v", keyID, data, participants)

	// 创建签名会话
	if err := store.CreateSignSession(keyID, data, participants); err != nil {
		log.Printf("创建签名会话失败: %v", err)
		return fmt.Errorf("创建签名会话失败: %w", err)
	}

	// 向参与方发送邀请
	invite := Message{
		Type: SignInviteMsg,
		Payload: SignInvitePayload{
			KeyID:        keyID,
			Data:         data,
			Participants: participants,
		},
	}

	for _, userID := range participants {
		invite.UserID = userID
		if err := SendMessageToUser(store, userID, invite); err != nil {
			log.Printf("向用户 %s 发送签名邀请失败: %v", userID, err)
			// 继续发送给其他参与者，不中断流程
		}
	}

	return nil
}

// HandleSignResponse 处理签名响应
// 解析响应载荷，更新会话状态，并检查是否有足够的参与方同意进行签名
// 如果有足够的参与方同意，则向他们发送签名参数
//
// 参数:
//   - store: 存储接口，用于管理会话状态和客户端连接
//   - msg: 包含签名响应数据的消息
//
// 返回:
//   - error: 如果处理过程中出现错误，返回相应错误；否则返回nil
func HandleSignResponse(store Storage, msg Message) error {
	var payload SignResponsePayload
	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		log.Printf("序列化签名响应载荷失败: %v", err)
		return fmt.Errorf("序列化载荷失败: %w", err)
	}

	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		log.Printf("解析签名响应载荷失败: %v", err)
		return fmt.Errorf("解析载荷失败: %w", err)
	}

	userID := msg.UserID
	keyID := payload.KeyID
	response := payload.Response

	log.Printf("收到签名响应 UserID: %s, KeyID: %s, 同意: %v", userID, keyID, response)

	// 更新会话状态
	err = store.UpdateSignSession(keyID, func(session *SignSession) {
		session.Responses[userID] = response
	})
	if err != nil {
		log.Printf("更新签名会话失败: %v", err)
		return fmt.Errorf("更新签名会话失败: %w", err)
	}

	// 获取会话信息
	session, exists := store.GetSignSession(keyID)
	if !exists {
		log.Printf("找不到签名会话: %s", keyID)
		return fmt.Errorf("找不到签名会话: %s", keyID)
	}

	// 检查是否所有参与方都已响应并同意
	allAgreed := true
	agreedCount := 0
	agreedUsers := []string{}

	for _, p := range session.Participants {
		if agreed, responded := session.Responses[p]; responded && agreed {
			agreedCount++
			agreedUsers = append(agreedUsers, p)
		} else if responded && !agreed {
			allAgreed = false
		}
	}

	// 假设阈值至少需要2人同意（实际应该从密钥配置中获取）
	// 如果同意人数达到阈值，发送签名参数
	minThreshold := 2 // 这里应该从密钥配置或会话中获取实际阈值
	if agreedCount >= minThreshold {
		log.Printf("足够的参与方同意签名 KeyID: %s, 同意的参与方: %v", keyID, agreedUsers)
		return sendSignParamsToParticipants(store, keyID, agreedUsers)
	} else if len(session.Responses) == len(session.Participants) && !allAgreed {
		log.Printf("有参与方拒绝签名 KeyID: %s", keyID)
		// 通知协调方签名取消
		return notifySignFailed(store, keyID)
	}

	return nil
}

// HandleSignResult 处理签名结果
// 解析结果载荷，保存签名结果，并将结果转发给协调方
//
// 参数:
//   - store: 存储接口，用于管理会话状态和客户端连接
//   - msg: 包含签名结果数据的消息
//
// 返回:
//   - error: 如果处理过程中出现错误，返回相应错误；否则返回nil
func HandleSignResult(store Storage, msg Message) error {
	var payload SignResultPayload
	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		log.Printf("序列化签名结果载荷失败: %v", err)
		return fmt.Errorf("序列化载荷失败: %w", err)
	}

	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		log.Printf("解析签名结果载荷失败: %v", err)
		return fmt.Errorf("解析载荷失败: %w", err)
	}

	userID := msg.UserID
	keyID := payload.KeyID
	signature := payload.Signature

	log.Printf("收到签名结果 UserID: %s, KeyID: %s", userID, keyID)

	// 更新会话状态
	err = store.UpdateSignSession(keyID, func(session *SignSession) {
		session.Results[userID] = signature
	})
	if err != nil {
		log.Printf("更新签名会话失败: %v", err)
		return fmt.Errorf("更新签名会话失败: %w", err)
	}

	// 将签名结果转发给协调方
	return forwardSignatureToCoordinator(store, keyID, signature)
}

// ExtractParticipantIndices 从参与者用户ID列表提取索引
//
// 参数:
//   - participants: 参与者用户ID列表，格式假设为 "userX" 其中X是数字索引
//
// 返回:
//   - string: 以逗号分隔的用户索引字符串
//   - error: 如果解析过程中出现错误，返回相应错误；否则返回nil
func ExtractParticipantIndices(participants []string) (string, error) {
	var indices []string
	for _, p := range participants {
		// 假设用户ID格式为"userX"其中X是数字索引
		index := strings.TrimPrefix(p, "user")
		if index == p {
			return "", fmt.Errorf("无效的用户ID格式: %s", p)
		}
		indices = append(indices, index)
	}
	return strings.Join(indices, ","), nil
}

// sendSignParamsToParticipants 向参与方发送签名参数
// 获取会话信息，并向每个同意参与签名的用户发送其特定的参数
//
// 参数:
//   - store: 存储接口，用于获取会话和客户端信息
//   - keyID: 密钥标识符
//   - agreedUsers: 同意参与签名的用户列表
//
// 返回:
//   - error: 如果处理过程中出现错误，返回相应错误；否则返回nil
func sendSignParamsToParticipants(store Storage, keyID string, agreedUsers []string) error {
	session, exists := store.GetSignSession(keyID)
	if !exists {
		log.Printf("找不到签名会话: %s", keyID)
		return fmt.Errorf("找不到签名会话: %s", keyID)
	}
	data := session.Data

	// 提取参与方索引
	participantsStr, err := ExtractParticipantIndices(agreedUsers)
	if err != nil {
		log.Printf("提取参与方索引失败: %v", err)
		return fmt.Errorf("提取参与方索引失败: %w", err)
	}

	// 为每个同意的参与方发送签名参数
	for _, userID := range agreedUsers {
		// 获取用户的密钥分享
		shareJSON, exists := store.GetUserShare(userID, keyID)
		if !exists {
			log.Printf("找不到用户密钥分享 UserID: %s, KeyID: %s", userID, keyID)
			continue
		}

		params := Message{
			Type:   SignParamsMsg,
			UserID: userID,
			Payload: SignParamsPayload{
				KeyID:        keyID,
				Data:         data,
				Participants: participantsStr,
				ShareJSON:    shareJSON,
			},
		}

		if err := SendMessageToUser(store, userID, params); err != nil {
			log.Printf("向用户 %s 发送签名参数失败: %v", userID, err)
			// 继续发送给其他参与者，不中断流程
		}
	}

	return nil
}

// notifySignFailed 通知协调方签名失败
// 查找协调方角色的用户，并发送签名失败的通知
//
// 参数:
//   - store: 存储接口，用于查找协调方用户
//   - keyID: 密钥标识符
//
// 返回:
//   - error: 如果处理过程中出现错误，返回相应错误；否则返回nil
func notifySignFailed(store Storage, keyID string) error {
	// 找出协调方
	var coordinator string
	clients := store.GetAllClients()
	for userID := range clients {
		if role, exists := store.GetClientRole(userID); exists && role == "coordinator" {
			coordinator = userID
			break
		}
	}

	if coordinator == "" {
		log.Printf("找不到协调方用户")
		return fmt.Errorf("找不到协调方用户")
	}

	// 发送通知
	notification := Message{
		Type:   SignCompleteMsg,
		UserID: coordinator,
		Payload: map[string]interface{}{
			"key_id": keyID,
			"status": "failed",
			"reason": "有参与方拒绝签名",
		},
	}

	return SendMessageToUser(store, coordinator, notification)
}

// forwardSignatureToCoordinator 将签名结果转发给协调方
// 查找协调方角色的用户，并发送签名结果的通知
//
// 参数:
//   - store: 存储接口，用于查找协调方用户
//   - keyID: 密钥标识符
//   - signature: 生成的签名
//
// 返回:
//   - error: 如果处理过程中出现错误，返回相应错误；否则返回nil
func forwardSignatureToCoordinator(store Storage, keyID string, signature string) error {
	// 找出协调方
	var coordinator string
	clients := store.GetAllClients()
	for userID := range clients {
		if role, exists := store.GetClientRole(userID); exists && role == "coordinator" {
			coordinator = userID
			break
		}
	}

	if coordinator == "" {
		log.Printf("找不到协调方用户")
		return fmt.Errorf("找不到协调方用户")
	}

	// 发送签名结果
	notification := Message{
		Type:   SignCompleteMsg,
		UserID: coordinator,
		Payload: map[string]interface{}{
			"key_id":    keyID,
			"status":    "success",
			"signature": signature,
		},
	}

	return SendMessageToUser(store, coordinator, notification)
}
