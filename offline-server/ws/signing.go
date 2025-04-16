package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// HandleSignRequest 处理签名请求
// 解析请求载荷，创建签名会话，并向所有参与方发送邀请
//
// 参数:
//   - store: 存储接口，用于管理会话状态和客户端连接
//   - msg: 包含签名请求数据的消息
//
// 返回:
//   - 如果处理过程中出现错误，返回相应错误；否则返回nil
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
	accountAddr := payload.AccountAddr
	var participants []string

	// 如果未指定参与者，则使用密钥生成时的参与者
	if len(payload.Participants) > 0 {
		participants = payload.Participants
	} else {
		// 从密钥生成会话中获取参与者列表
		keygenSession, exists := store.GetKeyGenSession(keyID)
		if !exists {
			log.Printf("找不到密钥生成会话: %s", keyID)
			return fmt.Errorf("找不到密钥生成会话: %s", keyID)
		}
		participants = keygenSession.Participants
	}

	log.Printf("收到签名请求 KeyID: %s, 数据: %s, 账户地址: %s, 参与方: %v", keyID, data, accountAddr, participants)

	// 创建签名会话
	if err := store.CreateSignSession(keyID, data, participants); err != nil {
		log.Printf("创建签名会话失败: %v", err)
		return fmt.Errorf("创建签名会话失败: %w", err)
	}

	// 向参与方发送邀请
	for i, userID := range participants {
		partIndex := i + 1 // 参与者索引从1开始
		invite := Message{
			Type:   SignInviteMsg,
			UserID: userID,
			Payload: SignInvitePayload{
				KeyID:        keyID,
				Data:         data,
				AccountAddr:  accountAddr,
				PartIndex:    partIndex,
				Participants: participants,
			},
		}

		if err := SendMessageToUser(store, userID, invite); err != nil {
			log.Printf("向用户 %s 发送邀请失败: %v", userID, err)
			// 继续发送给其他参与者，不中断流程
		}
	}

	return nil
}

// HandleSignResponse 处理签名响应
// 解析响应载荷，更新会话状态，并检查是否所有参与方都已响应
// 如果足够多的参与方同意，则向他们发送签名参数
//
// 参数:
//   - store: 存储接口，用于管理会话状态和客户端连接
//   - msg: 包含签名响应数据的消息
//
// 返回:
//   - 如果处理过程中出现错误，返回相应错误；否则返回nil
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
	partIndex := payload.PartIndex
	response := payload.Response

	log.Printf("收到签名响应 UserID: %s, KeyID: %s, 分片索引: %d, 同意: %v", userID, keyID, partIndex, response)

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

	// 计算同意参与签名的用户数量和用户列表
	agreedCount := 0
	var agreedUsers []string
	for userID, agreed := range session.Responses {
		if agreed {
			agreedCount++
			agreedUsers = append(agreedUsers, userID)
		}
	}

	// 获取密钥生成会话信息，以确定阈值
	keygenSession, exists := store.GetKeyGenSession(keyID)
	if !exists {
		log.Printf("找不到密钥生成会话: %s", keyID)
		return fmt.Errorf("找不到密钥生成会话: %s", keyID)
	}

	// 使用密钥生成会话中的阈值
	minThreshold := keygenSession.Threshold

	// 如果同意人数达到阈值，发送签名参数
	allResponded := len(session.Responses) == len(session.Participants)
	allAgreed := agreedCount == len(session.Participants)

	if agreedCount >= minThreshold {
		log.Printf("足够的参与方同意签名 KeyID: %s, 同意的参与方: %v", keyID, agreedUsers)
		return sendSignParamsToParticipants(store, keyID, agreedUsers)
	} else if allResponded && !allAgreed {
		log.Printf("有参与方拒绝签名 KeyID: %s", keyID)
		// 通知协调方签名取消
		return notifySignFailed(store, keyID)
	}

	return nil
}

// HandleSignResult 处理签名结果
// 解析结果载荷，保存签名结果，并检查是否有足够的结果合并最终签名
//
// 参数:
//   - store: 存储接口，用于管理会话状态和客户端连接
//   - msg: 包含签名结果数据的消息
//
// 返回:
//   - 如果处理过程中出现错误，返回相应错误；否则返回nil
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
	partIndex := payload.PartIndex
	signature := payload.Signature

	log.Printf("收到签名结果 UserID: %s, KeyID: %s, 分片索引: %d", userID, keyID, partIndex)

	// 更新会话状态
	err = store.UpdateSignSession(keyID, func(session *SignSession) {
		session.Results[userID] = signature
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

	// 如果有需要，这里可以实现签名结果合并逻辑
	// 简化起见，直接使用第一个收到的签名结果作为最终签名
	if len(session.Results) > 0 {
		// 获取第一个签名结果
		var finalSignature string
		for _, sig := range session.Results {
			finalSignature = sig
			break
		}

		log.Printf("已收集足够的签名结果 KeyID: %s", keyID)
		return forwardSignatureToCoordinator(store, keyID, finalSignature)
	}

	return nil
}

// sendSignParamsToParticipants 向参与方发送签名参数
// 获取会话信息和用户分享，并向每个参与者发送其特定的参数
//
// 参数:
//   - store: 存储接口，用于获取会话和客户端信息
//   - keyID: 密钥标识符
//   - participants: 参与签名的用户ID列表
//
// 返回:
//   - 如果处理过程中出现错误，返回相应错误；否则返回nil
func sendSignParamsToParticipants(store Storage, keyID string, participants []string) error {
	session, exists := store.GetSignSession(keyID)
	if !exists {
		log.Printf("找不到签名会话: %s", keyID)
		return fmt.Errorf("找不到签名会话: %s", keyID)
	}

	// 定义消息发送间隔，避免消息拥塞
	const messageSendDelay = 500 * time.Millisecond

	// 生成参与者索引字符串，如 "1,2,3"
	participantIndices := make([]string, 0, len(participants))
	participantIndicesMap := make(map[string]int)

	for i, userID := range session.Participants {
		for _, p := range participants {
			if p == userID {
				partIndex := i + 1 // 索引从1开始
				participantIndices = append(participantIndices, fmt.Sprintf("%d", partIndex))
				participantIndicesMap[userID] = partIndex
				break
			}
		}
	}

	participantsStr := strings.Join(participantIndices, ",")

	// 为每个参与方发送签名参数
	for _, userID := range participants {
		partIndex := participantIndicesMap[userID]
		shareJSON, exists := store.GetUserShare(userID, keyID)
		if !exists {
			log.Printf("找不到用户 %s 的密钥分享: %s", userID, keyID)
			continue
		}

		params := Message{
			Type:   SignParamsMsg,
			UserID: userID,
			Payload: SignParamsPayload{
				KeyID:        keyID,
				Data:         session.Data,
				PartIndex:    partIndex,
				Participants: participantsStr,
				ShareJSON:    shareJSON,
			},
		}

		if err := SendMessageToUser(store, userID, params); err != nil {
			log.Printf("向用户 %s 发送签名参数失败: %v", userID, err)
			// 继续发送给其他参与者，不中断流程
		}

		// 添加短暂延迟，避免消息拥塞
		time.Sleep(messageSendDelay)
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
//   - 如果处理过程中出现错误，返回相应错误；否则返回nil
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
			"reason": "有参与方拒绝签名或未达到签名阈值",
		},
	}

	return SendMessageToUser(store, coordinator, notification)
}

// forwardSignatureToCoordinator 将签名结果转发给协调方
// 查找协调方角色的用户，并发送签名结果
//
// 参数:
//   - store: 存储接口，用于查找协调方用户
//   - keyID: 密钥标识符
//   - signature: 生成的签名
//
// 返回:
//   - 如果处理过程中出现错误，返回相应错误；否则返回nil
func forwardSignatureToCoordinator(store Storage, keyID string, signature string) error {
	// 获取签名会话信息，以获取账户地址
	session, exists := store.GetSignSession(keyID)
	if !exists {
		log.Printf("找不到签名会话: %s", keyID)
		return fmt.Errorf("找不到签名会话: %s", keyID)
	}

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
			"key_id":       keyID,
			"status":       "success",
			"signature":    signature,
			"data":         session.Data,
			"account_addr": session.AccountAddr,
		},
	}

	return SendMessageToUser(store, coordinator, notification)
}
