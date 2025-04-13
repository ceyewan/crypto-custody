package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// HandleKeyGenRequest 处理密钥生成请求
// 解析请求载荷，创建密钥生成会话，并向所有参与方发送邀请
//
// 参数:
//   - store: 存储接口，用于管理会话状态和客户端连接
//   - msg: 包含密钥生成请求数据的消息
//
// 返回:
//   - 如果处理过程中出现错误，返回相应错误；否则返回nil
func HandleKeyGenRequest(store Storage, msg Message) error {
	var payload KeyGenRequestPayload
	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		log.Printf("序列化密钥生成请求载荷失败: %v", err)
		return fmt.Errorf("序列化载荷失败: %w", err)
	}

	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		log.Printf("解析密钥生成请求载荷失败: %v", err)
		return fmt.Errorf("解析载荷失败: %w", err)
	}

	keyID := payload.KeyID
	threshold := payload.Threshold
	participants := payload.Participants

	log.Printf("收到密钥生成请求 KeyID: %s, 阈值: %d, 参与方: %v", keyID, threshold, participants)

	// 创建密钥生成会话
	if err := store.CreateKeyGenSession(keyID, threshold, participants); err != nil {
		log.Printf("创建密钥生成会话失败: %v", err)
		return fmt.Errorf("创建密钥生成会话失败: %w", err)
	}

	// 向参与方发送邀请
	invite := Message{
		Type: KeyGenInviteMsg,
		Payload: KeyGenInvitePayload{
			KeyID:        keyID,
			Threshold:    threshold,
			Participants: participants,
		},
	}

	for _, userID := range participants {
		invite.UserID = userID
		if err := SendMessageToUser(store, userID, invite); err != nil {
			log.Printf("向用户 %s 发送邀请失败: %v", userID, err)
			// 继续发送给其他参与者，不中断流程
		}
	}

	return nil
}

// HandleKeyGenResponse 处理密钥生成响应
// 解析响应载荷，更新会话状态，并检查是否所有参与方都已响应
// 如果所有参与方都同意，则向他们发送密钥生成参数
//
// 参数:
//   - store: 存储接口，用于管理会话状态和客户端连接
//   - msg: 包含密钥生成响应数据的消息
//
// 返回:
//   - 如果处理过程中出现错误，返回相应错误；否则返回nil
func HandleKeyGenResponse(store Storage, msg Message) error {
	var payload KeyGenResponsePayload
	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		log.Printf("序列化密钥生成响应载荷失败: %v", err)
		return fmt.Errorf("序列化载荷失败: %w", err)
	}

	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		log.Printf("解析密钥生成响应载荷失败: %v", err)
		return fmt.Errorf("解析载荷失败: %w", err)
	}

	userID := msg.UserID
	keyID := payload.KeyID
	response := payload.Response

	log.Printf("收到密钥生成响应 UserID: %s, KeyID: %s, 同意: %v", userID, keyID, response)

	// 更新会话状态
	err = store.UpdateKeyGenSession(keyID, func(session *KeyGenSession) {
		session.Responses[userID] = response
	})
	if err != nil {
		log.Printf("更新密钥生成会话失败: %v", err)
		return fmt.Errorf("更新密钥生成会话失败: %w", err)
	}

	// 检查是否所有参与方都已响应且同意
	session, exists := store.GetKeyGenSession(keyID)
	if !exists {
		log.Printf("找不到密钥生成会话: %s", keyID)
		return fmt.Errorf("找不到密钥生成会话: %s", keyID)
	}

	allResponded := true
	allAgreed := true
	for _, p := range session.Participants {
		if agreed, responded := session.Responses[p]; !responded {
			allResponded = false
			break
		} else if !agreed {
			allAgreed = false
			break
		}
	}

	// 如果所有参与方都同意，发送密钥生成参数
	if allResponded && allAgreed {
		log.Printf("所有参与方都同意密钥生成 KeyID: %s", keyID)
		return sendKeyGenParamsToParticipants(store, keyID)
	} else if allResponded {
		log.Printf("有参与方拒绝密钥生成 KeyID: %s", keyID)
		// 通知协调方密钥生成取消
		return notifyKeyGenFailed(store, keyID)
	}

	return nil
}

// HandleKeyGenComplete 处理密钥生成完成消息
// 解析完成载荷，保存用户的密钥分享，并检查是否所有参与方都已完成
// 如果所有参与方都完成了密钥生成，则通知协调方生成成功
//
// 参数:
//   - store: 存储接口，用于管理会话状态和客户端连接
//   - msg: 包含密钥生成完成数据的消息
//
// 返回:
//   - 如果处理过程中出现错误，返回相应错误；否则返回nil
func HandleKeyGenComplete(store Storage, msg Message) error {
	var payload KeyGenCompletePayload
	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		log.Printf("序列化密钥生成完成载荷失败: %v", err)
		return fmt.Errorf("序列化载荷失败: %w", err)
	}

	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		log.Printf("解析密钥生成完成载荷失败: %v", err)
		return fmt.Errorf("解析载荷失败: %w", err)
	}

	userID := msg.UserID
	keyID := payload.KeyID
	shareJSON := payload.ShareJSON

	log.Printf("收到密钥生成完成 UserID: %s, KeyID: %s", userID, keyID)

	// 保存用户的密钥分享
	store.AddUserShare(userID, keyID, shareJSON)

	// 更新会话状态
	err = store.UpdateKeyGenSession(keyID, func(session *KeyGenSession) {
		session.Completed[userID] = true
	})
	if err != nil {
		log.Printf("更新密钥生成会话失败: %v", err)
		return fmt.Errorf("更新密钥生成会话失败: %w", err)
	}

	// 检查是否所有参与方都已完成密钥生成
	session, exists := store.GetKeyGenSession(keyID)
	if !exists {
		log.Printf("找不到密钥生成会话: %s", keyID)
		return fmt.Errorf("找不到密钥生成会话: %s", keyID)
	}

	allCompleted := true
	for _, p := range session.Participants {
		if !session.Completed[p] {
			allCompleted = false
			break
		}
	}

	// 如果所有参与方都完成了密钥生成，通知协调方
	if allCompleted {
		log.Printf("所有参与方都完成了密钥生成 KeyID: %s", keyID)
		return notifyKeyGenSuccess(store, keyID)
	}

	return nil
}

// sendKeyGenParamsToParticipants 向参与方发送密钥生成参数
// 获取会话信息，并向每个参与者发送其特定的参数
//
// 参数:
//   - store: 存储接口，用于获取会话和客户端信息
//   - keyID: 密钥标识符
//
// 返回:
//   - 如果处理过程中出现错误，返回相应错误；否则返回nil
func sendKeyGenParamsToParticipants(store Storage, keyID string) error {
	session, exists := store.GetKeyGenSession(keyID)
	if !exists {
		log.Printf("找不到密钥生成会话: %s", keyID)
		return fmt.Errorf("找不到密钥生成会话: %s", keyID)
	}

	threshold := session.Threshold
	totalParts := len(session.Participants)
	participants := session.Participants

	// 定义消息发送间隔，避免消息拥塞
	const messageSendDelay = 500 * time.Millisecond

	// 为每个参与方发送密钥生成参数
	for i, userID := range participants {
		partIndex := i + 1 // 索引从1开始
		outputFile := fmt.Sprintf("local-share%d.json", partIndex)

		params := Message{
			Type:   KeyGenParamsMsg,
			UserID: userID,
			Payload: KeyGenParamsPayload{
				KeyID:      keyID,
				Threshold:  threshold,
				TotalParts: totalParts,
				PartIndex:  partIndex,
				OutputFile: outputFile,
			},
		}

		if err := SendMessageToUser(store, userID, params); err != nil {
			log.Printf("向用户 %s 发送密钥生成参数失败: %v", userID, err)
			// 继续发送给其他参与者，不中断流程
		}

		// 添加短暂延迟，避免消息拥塞
		time.Sleep(messageSendDelay)
	}

	return nil
}

// notifyKeyGenFailed 通知协调方密钥生成失败
// 查找协调方角色的用户，并发送密钥生成失败的通知
//
// 参数:
//   - store: 存储接口，用于查找协调方用户
//   - keyID: 密钥标识符
//
// 返回:
//   - 如果处理过程中出现错误，返回相应错误；否则返回nil
func notifyKeyGenFailed(store Storage, keyID string) error {
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
		Type:   KeyGenConfirmMsg,
		UserID: coordinator,
		Payload: map[string]interface{}{
			"key_id": keyID,
			"status": "failed",
			"reason": "有参与方拒绝密钥生成",
		},
	}

	return SendMessageToUser(store, coordinator, notification)
}

// notifyKeyGenSuccess 通知协调方密钥生成成功
// 查找协调方角色的用户，并发送密钥生成成功的通知
//
// 参数:
//   - store: 存储接口，用于查找协调方用户
//   - keyID: 密钥标识符
//
// 返回:
//   - 如果处理过程中出现错误，返回相应错误；否则返回nil
func notifyKeyGenSuccess(store Storage, keyID string) error {
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
		Type:   KeyGenConfirmMsg,
		UserID: coordinator,
		Payload: map[string]interface{}{
			"key_id": keyID,
			"status": "success",
		},
	}

	return SendMessageToUser(store, coordinator, notification)
}
