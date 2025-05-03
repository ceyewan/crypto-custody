package ws

import (
	"encoding/json"
	"fmt"
	"log"

	"offline-server/storage"
	"offline-server/storage/model"
	mem_storage "offline-server/ws/storage"
)

// MessageHandler 消息处理器
// 负责处理各种WebSocket消息
type MessageHandler struct {
	shareStorage   storage.IShareStorage       // 私钥分片存储接口
	seStorage      storage.ISeStorage          // 安全芯片存储接口
	sessionManager *mem_storage.SessionManager // 会话管理器
}

// NewMessageHandler 创建新的消息处理器
func NewMessageHandler() *MessageHandler {
	return &MessageHandler{
		shareStorage:   storage.GetShareStorage(),
		seStorage:      storage.GetSeStorage(),
		sessionManager: mem_storage.GetSessionManager(),
	}
}

// ProcessMessage 处理收到的WebSocket消息
func (h *MessageHandler) ProcessMessage(msgType MessageType, rawMessage []byte, sender *Client) error {
	// 根据消息类型分发处理
	switch msgType {
	case MsgKeyGenRequest:
		var msg KeyGenRequestMessage
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			return fmt.Errorf("解析密钥生成请求消息失败: %w", err)
		}
		return h.handleKeyGenRequest(msg, sender)

	case MsgKeyGenResponse:
		var msg KeyGenResponseMessage
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			return fmt.Errorf("解析密钥生成响应消息失败: %w", err)
		}
		return h.handleKeyGenResponse(msg, sender)

	case MsgKeyGenResult:
		var msg KeyGenResultMessage
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			return fmt.Errorf("解析密钥生成结果消息失败: %w", err)
		}
		return h.handleKeyGenResult(msg, sender)

	case MsgSignRequest:
		var msg SignRequestMessage
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			return fmt.Errorf("解析签名请求消息失败: %w", err)
		}
		return h.handleSignRequest(msg, sender)

	case MsgSignResponse:
		var msg SignResponseMessage
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			return fmt.Errorf("解析签名响应消息失败: %w", err)
		}
		return h.handleSignResponse(msg, sender)

	case MsgSignResult:
		var msg SignResultMessage
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			return fmt.Errorf("解析签名结果消息失败: %w", err)
		}
		return h.handleSignResult(msg, sender)

	default:
		return fmt.Errorf("不支持的消息类型: %s", msgType)
	}
}

// handleKeyGenRequest 处理密钥生成请求
func (h *MessageHandler) handleKeyGenRequest(msg KeyGenRequestMessage, sender *Client) error {
	// 直接从消息结构体获取需要的字段
	sessionKey := msg.SessionKey
	threshold := msg.Threshold
	totalParts := msg.TotalParts
	participants := msg.Participants

	log.Printf("收到密钥生成请求 SessionKey: %s, 阈值: %d, 总分片数: %d, 参与者: %v",
		sessionKey, threshold, totalParts, participants)

	// 创建密钥生成会话
	if err := h.sessionManager.CreateKeyGenSession(sessionKey, sender.Username(), threshold, totalParts, participants); err != nil {
		return fmt.Errorf("创建密钥生成会话失败: %w", err)
	}

	// 获取 totalParts 数量的安全芯片 SeID
	chips, err := h.seStorage.GetRandomSeIds(totalParts)
	if err != nil {
		return fmt.Errorf("获取安全芯片标识符失败: %w", err)
	}

	// 更新密钥生成会话的安全芯片标识符
	h.sessionManager.GetKeyGenSession(sessionKey).Chips = chips

	// 向所有参与方发送邀请
	for i, participant := range participants {
		// 准备邀请消息
		inviteMsg := KeyGenInviteMessage{
			BaseMessage:  BaseMessage{Type: MsgKeyGenInvite},
			SessionKey:   sessionKey,
			Coordinator:  sender.Username(),
			Threshold:    threshold,
			TotalParts:   totalParts,
			PartIndex:    i + 1,    // 索引从1开始
			SeID:         chips[i], // 安全芯片标识符
			Participants: participants,
		}

		// 发送邀请
		client, exists := sender.Hub().GetClient(participant)
		if !exists {
			log.Printf("参与方 %s 不在线，无法发送邀请", participant)
			continue
		}

		if err := client.SendMessage(inviteMsg); err != nil {
			log.Printf("向参与方 %s 发送邀请失败: %v", participant, err)
		}
	}

	h.sessionManager.GetKeyGenSession(sessionKey).Status = model.StatusInvited

	return nil
}

// handleKeyGenResponse 处理密钥生成响应
func (h *MessageHandler) handleKeyGenResponse(msg KeyGenResponseMessage, sender *Client) error {
	// 直接从消息结构体获取字段
	sessionKey := msg.SessionKey
	partIndex := msg.PartIndex
	cpic := msg.CPIC
	accept := msg.Accept
	reason := msg.Reason

	log.Printf("收到密钥生成响应 SessionKey: %s, 索引: %d, 接受状态: %v",
		sessionKey, partIndex, accept)

	// 获取会话
	session := h.sessionManager.GetKeyGenSession(sessionKey)

	// 验证芯片标识符是否匹配
	se, err := h.seStorage.GetSeBySeId(session.Chips[partIndex])
	if err != nil {
		return fmt.Errorf("验证安全芯片标识符失败: %w", err)
	}
	if se.CPIC != cpic {
		return fmt.Errorf("安全芯片标识符不匹配: %s != %s", se.CPIC, cpic)
	}

	// 更新会话状态
	if accept {
		// 接受邀请
		session.Responses[partIndex-1] = string(model.ParticipantAccepted)

		// 检查是否所有参与方都已接受，统计 session.Responses 是否全为 Accepted
		flag := true
		for _, status := range session.Responses {
			if status != string(model.ParticipantAccepted) {
				flag = false
			}
		}

		if flag {
			// 向所有参与方发送参数
			for i, participant := range session.Participants {
				// 准备参数消息
				paramsMsg := KeyGenParamsMessage{
					BaseMessage: BaseMessage{Type: MsgKeyGenParams},
					SessionKey:  sessionKey,
					Threshold:   session.Threshold,
					TotalParts:  len(session.Participants),
					PartIndex:   i + 1,
					FileName:    fmt.Sprintf("%s_%d.json", sessionKey, i+1),
				}

				// 发送参数
				client, exists := sender.Hub().GetClient(participant)
				if !exists {
					log.Printf("参与方 %s 不在线，无法发送参数", participant)
					continue
				}

				if err := client.SendMessage(paramsMsg); err != nil {
					log.Printf("向参与方 %s 发送参数失败: %v", participant, err)
				}
			}
		}
	} else {
		// 拒绝邀请
		session.Responses[partIndex-1] = string(model.ParticipantRejected)

		// 通知发起者有参与方拒绝
		initiator := session.Initiator
		rejectMsg := ErrorMessage{
			BaseMessage: BaseMessage{Type: MsgError},
			Message:     fmt.Sprintf("参与方 %s 拒绝了密钥生成邀请", sender.Username()),
			Details:     reason,
		}

		// 发送拒绝通知
		client, exists := sender.Hub().GetClient(initiator)
		if exists {
			if err := client.SendMessage(rejectMsg); err != nil {
				log.Printf("向发起者 %s 发送拒绝通知失败: %v", initiator, err)
			}
		}
	}

	return nil
}

// handleKeyGenResult 处理密钥生成结果
func (h *MessageHandler) handleKeyGenResult(msg KeyGenResultMessage, sender *Client) error {
	// 直接从消息结构体获取字段
	success := msg.Success
	if !success {
		return fmt.Errorf("密钥生成失败: %s", msg.Message)
	}
	sessionKey := msg.SessionKey
	partIndex := msg.PartIndex
	address := msg.Address
	cpic := msg.CPIC
	encryptedShard := msg.EncryptedShard

	log.Printf("收到密钥生成结果 SessionKey: %s, 索引: %d", sessionKey, partIndex)

	// 保存私钥分片
	if err := h.shareStorage.CreateEthereumKeyShard(sender.Username(), address, cpic, encryptedShard, partIndex); err != nil {
		return fmt.Errorf("保存密钥分片失败: %w", err)
	}

	// 更新会话状态
	session := h.sessionManager.GetKeyGenSession(sessionKey)

	// 标记该部分已完成
	session.Responses[partIndex-1] = string(model.ParticipantCompleted)

	// 检查是否所有参与方都已完成
	allCompleted := true
	for _, status := range session.Responses {
		if status != string(model.ParticipantCompleted) {
			allCompleted = false
			break
		}
	}

	if allCompleted {
		// 更新会话状态为完成
		session.Status = model.StatusCompleted

		// 通知发起者密钥生成已完成
		initiator := session.Initiator
		confirmMsg := KeyGenCompleteMessage{
			BaseMessage: BaseMessage{Type: MsgKeyGenComplete},
			SessionKey:  sessionKey,
			Address:     address,
			Success:     true,
			Message:     "密钥生成已完成",
		}

		// 发送确认消息
		client, exists := sender.Hub().GetClient(initiator)
		if exists {
			if err := client.SendMessage(confirmMsg); err != nil {
				log.Printf("向发起者 %s 发送确认消息失败: %v", initiator, err)
			}
		}
	}

	return nil
}

// handleSignRequest 处理签名请求
func (h *MessageHandler) handleSignRequest(msg SignRequestMessage, sender *Client) error {
	// 直接从消息结构体获取字段
	sessionKey := msg.SessionKey
	data := msg.Data
	address := msg.Address
	participants := msg.Participants

	log.Printf("收到签名请求 SessionKey: %s, 数据: %s, 账户地址: %s, 参与者: %v",
		sessionKey, data, address, participants)

	// 创建签名会话
	session, err := h.sessionManager.CreateSignSession(sessionKey, sender.Username(), data, address, participants)
	if err != nil {
		return fmt.Errorf("创建签名会话失败: %w", err)
	}

	// 为每个参与者分配安全芯片ID
	chips, err := h.seStorage.GetRandomSeIds(len(participants))
	if err != nil {
		return fmt.Errorf("获取安全芯片标识符失败: %w", err)
	}
	session.Chips = chips

	// 向所有参与方发送邀请
	for i, participant := range participants {
		// 准备邀请消息
		inviteMsg := SignInviteMessage{
			BaseMessage:  BaseMessage{Type: MsgSignInvite},
			SessionKey:   sessionKey,
			Data:         data,
			Address:      address,
			PartIndex:    i + 1,
			SeID:         chips[i], // 安全芯片标识符
			Participants: participants,
		}

		// 发送邀请
		client, exists := sender.Hub().GetClient(participant)
		if !exists {
			log.Printf("参与方 %s 不在线，无法发送邀请", participant)
			continue
		}

		if err := client.SendMessage(inviteMsg); err != nil {
			log.Printf("向参与方 %s 发送邀请失败: %v", participant, err)
		}
	}

	// 更新会话状态为已邀请
	session.Status = model.StatusInvited

	return nil
}

// handleSignResponse 处理签名响应
func (h *MessageHandler) handleSignResponse(msg SignResponseMessage, sender *Client) error {
	// 直接从消息结构体获取字段
	sessionKey := msg.SessionKey
	partIndex := msg.PartIndex
	cpic := msg.CPIC
	accept := msg.Accept
	reason := msg.Reason

	// 获取会话
	session := h.sessionManager.GetSignSession(sessionKey)

	// 验证芯片标识符是否匹配
	se, err := h.seStorage.GetSeBySeId(session.Chips[partIndex-1])
	if err != nil {
		return fmt.Errorf("验证安全芯片标识符失败: %w", err)
	}
	if se.CPIC != cpic {
		return fmt.Errorf("安全芯片标识符不匹配: %s != %s", se.CPIC, cpic)
	}

	log.Printf("收到签名响应 SessionKey: %s, 索引: %d, 接受状态: %v",
		sessionKey, partIndex, accept)

	// 更新会话状态
	if accept {
		// 接受邀请
		session.Responses[partIndex-1] = string(model.ParticipantAccepted)

		// 检查是否所有参与方都已接受
		allAccepted := true
		for _, status := range session.Responses {
			if status != string(model.ParticipantAccepted) {
				allAccepted = false
				break
			}
		}

		if allAccepted {
			// 向所有参与方发送参数
			for _, participant := range session.Participants {
				// 获取该参与者的密钥分片数据
				encryptedShard, err := h.shareStorage.GetEthereumKeyShard(participant, session.AccountAddr)
				if err != nil {
					log.Printf("获取参与方 %s 的密钥分片失败: %v", participant, err)
					continue
				}

				// 生成签名参与者列表(索引集合)
				parties := make([]int, len(session.Participants))
				for j := range session.Participants {
					parties[j] = j + 1
				}

				// 准备参数消息
				paramsMsg := SignParamsMessage{
					BaseMessage:    BaseMessage{Type: MsgSignParams},
					SessionKey:     sessionKey,
					Data:           session.Data,
					Address:        session.AccountAddr,
					PartIndex:      encryptedShard.ShardIndex,
					FileName:       fmt.Sprintf("%s_%d.json", sessionKey, encryptedShard.ShardIndex),
					Parties:        fmt.Sprintf("%v", parties),
					EncryptedShard: encryptedShard.PrivateShard,
				}

				// 发送参数
				client, exists := sender.Hub().GetClient(participant)
				if !exists {
					log.Printf("参与方 %s 不在线，无法发送参数", participant)
					continue
				}

				if err := client.SendMessage(paramsMsg); err != nil {
					log.Printf("向参与方 %s 发送参数失败: %v", participant, err)
				}
			}
		}
	} else {
		// 拒绝邀请
		session.Responses[partIndex-1] = string(model.ParticipantRejected)

		// 通知发起者有参与方拒绝
		initiator := session.Initiator
		rejectMsg := ErrorMessage{
			BaseMessage: BaseMessage{Type: MsgError},
			Message:     fmt.Sprintf("参与方 %s 拒绝了签名邀请", sender.Username()),
			Details:     reason,
		}

		// 发送拒绝通知
		client, exists := sender.Hub().GetClient(initiator)
		if exists {
			if err := client.SendMessage(rejectMsg); err != nil {
				log.Printf("向发起者 %s 发送拒绝通知失败: %v", initiator, err)
			}
		}
	}

	return nil
}

// handleSignResult 处理签名结果
func (h *MessageHandler) handleSignResult(msg SignResultMessage, sender *Client) error {
	// 直接从消息结构体获取字段
	sessionKey := msg.SessionKey
	partIndex := msg.PartIndex
	signature := msg.Signature

	log.Printf("收到签名结果 SessionKey: %s, 索引: %d", sessionKey, partIndex)

	// 获取会话
	session := h.sessionManager.GetSignSession(sessionKey)

	// 更新会话状态
	session.Responses[partIndex-1] = string(model.ParticipantCompleted)
	session.Signature = signature

	// 检查是否所有参与方都已完成
	allCompleted := true
	for _, status := range session.Responses {
		if status != string(model.ParticipantCompleted) {
			allCompleted = false
			break
		}
	}

	if allCompleted {

		// 通知发起者签名已完成
		initiator := session.Initiator
		completeMsg := SignCompleteMessage{
			BaseMessage: BaseMessage{Type: MsgSignComplete},
			SessionKey:  sessionKey,
			Signature:   signature,
			Success:     true,
			Message:     "签名已完成",
		}

		// 发送完成消息
		client, exists := sender.Hub().GetClient(initiator)
		if exists {
			if err := client.SendMessage(completeMsg); err != nil {
				log.Printf("向发起者 %s 发送完成消息失败: %v", initiator, err)
			}
		}
	}

	return nil
}
