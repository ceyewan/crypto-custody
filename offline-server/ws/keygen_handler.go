package ws

import (
	"encoding/json"
	"fmt"

	"offline-server/clog"
	"offline-server/storage"
	"offline-server/storage/model"
	mem_storage "offline-server/ws/storage"
)

// KeyGenHandler 密钥生成消息处理器
type KeyGenHandler struct {
	shareStorage   storage.IShareStorage       // 私钥分片存储接口
	seStorage      storage.ISeStorage          // 安全芯片存储接口
	sessionManager *mem_storage.SessionManager // 会话管理器
}

// NewKeyGenHandler 创建密钥生成消息处理器
func NewKeyGenHandler(shareStorage storage.IShareStorage, seStorage storage.ISeStorage, sessionManager *mem_storage.SessionManager) *KeyGenHandler {
	handler := &KeyGenHandler{
		shareStorage:   shareStorage,
		seStorage:      seStorage,
		sessionManager: sessionManager,
	}

	clog.Debug("创建密钥生成消息处理器实例")
	return handler
}

// ProcessMessage 处理密钥生成相关消息
func (h *KeyGenHandler) ProcessMessage(msgType MessageType, rawMessage []byte, sender *Client) error {
	clog.Debug("处理密钥生成消息",
		clog.String("msg_type", string(msgType)),
		clog.String("username", sender.GetUserName()),
		clog.Int("msg_size", len(rawMessage)))

	// 根据消息类型分发处理
	switch msgType {
	case MsgKeyGenRequest:
		var msg KeyGenRequestMessage
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			clog.Error("解析密钥生成请求消息失败",
				clog.Err(err),
				clog.String("username", sender.GetUserName()))
			return fmt.Errorf("解析密钥生成请求消息失败: %w", err)
		}
		return h.handleKeyGenRequest(msg, sender)

	case MsgKeyGenResponse:
		var msg KeyGenResponseMessage
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			clog.Error("解析密钥生成响应消息失败",
				clog.Err(err),
				clog.String("username", sender.GetUserName()))
			return fmt.Errorf("解析密钥生成响应消息失败: %w", err)
		}
		return h.handleKeyGenResponse(msg, sender)

	case MsgKeyGenResult:
		var msg KeyGenResultMessage
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			clog.Error("解析密钥生成结果消息失败",
				clog.Err(err),
				clog.String("username", sender.GetUserName()))
			return fmt.Errorf("解析密钥生成结果消息失败: %w", err)
		}
		return h.handleKeyGenResult(msg, sender)

	default:
		clog.Error("不支持的密钥生成消息类型",
			clog.String("msg_type", string(msgType)),
			clog.String("username", sender.GetUserName()))
		return fmt.Errorf("不支持的密钥生成消息类型: %s", msgType)
	}
}

// notifySessionFailure 通知会话发起者失败
func (h *KeyGenHandler) notifySessionFailure(sessionKey string, sender *Client, message string, details string) {
	session := h.sessionManager.GetKeyGenSession(sessionKey)
	if session == nil {
		clog.Warn("无法获取会话信息，无法发送失败通知",
			clog.String("session_key", sessionKey))
		return
	}

	// 更新会话状态为失败
	session.Status = model.StatusFailed

	// 通知发起者失败
	initiator := session.Initiator
	failureMsg := ErrorMessage{
		BaseMessage: BaseMessage{Type: MsgError},
		Message:     message,
		Details:     details,
	}

	client, exists := sender.Hub().GetClient(initiator)
	if exists {
		if err := client.SendMessage(failureMsg); err != nil {
			clog.Error("向发起者发送失败通知失败",
				clog.Err(err),
				clog.String("initiator", initiator),
				clog.String("session_key", sessionKey))
		} else {
			clog.Debug("向发起者发送失败通知成功",
				clog.String("initiator", initiator),
				clog.String("session_key", sessionKey),
				clog.String("message", message))
		}
	} else {
		clog.Warn("发起者不在线，无法发送失败通知",
			clog.String("initiator", initiator),
			clog.String("session_key", sessionKey))
	}
}

// handleKeyGenRequest 处理密钥生成请求
func (h *KeyGenHandler) handleKeyGenRequest(msg KeyGenRequestMessage, sender *Client) error {
	// 直接从消息结构体获取需要的字段
	sessionKey := msg.SessionKey
	threshold := msg.Threshold
	totalParts := msg.TotalParts
	participants := msg.Participants

	clog.Info("收到密钥生成请求",
		clog.String("session_key", sessionKey))

	clog.Debug("密钥生成请求详情",
		clog.String("session_key", sessionKey),
		clog.Int("threshold", threshold),
		clog.Int("total_parts", totalParts),
		clog.Any("participants", participants))

	// 创建密钥生成会话
	if err := h.sessionManager.CreateKeyGenSession(sessionKey, sender.GetUserName(), threshold, totalParts, participants); err != nil {
		clog.Error("创建密钥生成会话失败",
			clog.Err(err),
			clog.String("session_key", sessionKey))

		// 直接向发起者发送失败消息
		failureMsg := ErrorMessage{
			BaseMessage: BaseMessage{Type: MsgError},
			Message:     "创建密钥生成会话失败",
			Details:     err.Error(),
		}
		if sendErr := sender.SendMessage(failureMsg); sendErr != nil {
			clog.Error("向发起者发送失败通知失败",
				clog.Err(sendErr),
				clog.String("username", sender.GetUserName()))
		}

		return fmt.Errorf("创建密钥生成会话失败: %w", err)
	}

	// 获取 totalParts 数量的安全芯片 SeID
	chips, err := h.seStorage.GetRandomSeIds(totalParts)
	if err != nil {
		clog.Error("获取安全芯片标识符失败",
			clog.Err(err),
			clog.Int("requested_count", totalParts))

		// 获取会话并更新状态为失败
		session := h.sessionManager.GetKeyGenSession(sessionKey)
		if session != nil {
			session.Status = model.StatusFailed
		}

		// 通知发起者失败
		h.notifySessionFailure(sessionKey, sender, "获取安全芯片标识符失败", err.Error())

		return fmt.Errorf("获取安全芯片标识符失败: %w", err)
	}

	clog.Debug("获取到安全芯片标识符",
		clog.Int("chip_count", len(chips)),
		clog.Any("chips", chips))

	// 更新密钥生成会话的安全芯片标识符
	session := h.sessionManager.GetKeyGenSession(sessionKey)
	session.Chips = chips

	// 记录离线参与者
	offlineParticipants := []string{}

	// 向所有参与方发送邀请
	for i, participant := range participants {
		// 准备邀请消息
		inviteMsg := KeyGenInviteMessage{
			BaseMessage:  BaseMessage{Type: MsgKeyGenInvite},
			SessionKey:   sessionKey,
			Coordinator:  sender.GetUserName(),
			Threshold:    threshold,
			TotalParts:   totalParts,
			PartIndex:    i + 1,    // 索引从1开始
			SeID:         chips[i], // 安全芯片标识符
			Participants: participants,
		}

		// 发送邀请
		client, exists := sender.Hub().GetClient(participant)
		if !exists {
			clog.Warn("参与方不在线，无法发送邀请",
				clog.String("participant", participant),
				clog.String("session_key", sessionKey))
			offlineParticipants = append(offlineParticipants, participant)
			continue
		}

		if err := client.SendMessage(inviteMsg); err != nil {
			clog.Error("向参与方发送邀请失败",
				clog.Err(err),
				clog.String("participant", participant),
				clog.String("session_key", sessionKey))
			offlineParticipants = append(offlineParticipants, participant)
		}
	}

	// 检查是否有参与者离线
	if len(offlineParticipants) > 0 {
		clog.Error("部分参与者离线或发送邀请失败",
			clog.Any("offline_participants", offlineParticipants),
			clog.String("session_key", sessionKey))

		// 更新会话状态为失败
		session.Status = model.StatusFailed

		// 构建失败消息
		offlineMsg := fmt.Sprintf("以下参与者不在线或发送邀请失败: %v", offlineParticipants)
		h.notifySessionFailure(sessionKey, sender, "密钥生成初始化失败", offlineMsg)

		return fmt.Errorf("部分参与者离线或发送邀请失败: %v", offlineParticipants)
	}

	session.Status = model.StatusInvited
	clog.Info("密钥生成会话状态已更新为已邀请",
		clog.String("session_key", sessionKey),
		clog.String("status", string(model.StatusInvited)))

	return nil
}

// handleKeyGenResponse 处理密钥生成响应
func (h *KeyGenHandler) handleKeyGenResponse(msg KeyGenResponseMessage, sender *Client) error {
	// 直接从消息结构体获取字段
	sessionKey := msg.SessionKey
	partIndex := msg.PartIndex
	cpic := msg.CPIC
	accept := msg.Accept
	reason := msg.Reason

	clog.Info("收到密钥生成响应",
		clog.String("session_key", sessionKey),
		clog.String("participant", sender.GetUserName()),
		clog.Int("part_index", partIndex),
		clog.Bool("accept", accept))

	// 获取会话
	session := h.sessionManager.GetKeyGenSession(sessionKey)
	if session == nil {
		errMsg := fmt.Sprintf("找不到对应的密钥生成会话: %s", sessionKey)
		clog.Error(errMsg)
		return fmt.Errorf(errMsg)
	}

	// 如果会话状态已经是失败，直接返回
	if session.Status == model.StatusFailed {
		clog.Warn("会话已处于失败状态，忽略响应",
			clog.String("session_key", sessionKey),
			clog.String("participant", sender.GetUserName()))
		return nil
	}

	if !accept {
		// 拒绝邀请
		session.Responses[partIndex-1] = string(model.ParticipantRejected)
		clog.Info("参与方拒绝密钥生成邀请",
			clog.String("session_key", sessionKey),
			clog.String("participant", sender.GetUserName()),
			clog.Int("part_index", partIndex),
			clog.String("reason", reason))

		// 更新会话状态
		session.Status = model.StatusFailed
		clog.Info("密钥生成会话状态已更新为失败")

		// 通知发起者有参与方拒绝
		h.notifySessionFailure(sessionKey, sender,
			fmt.Sprintf("参与方 %s 拒绝了密钥生成邀请", sender.GetUserName()),
			reason)

		return nil
	}

	// 验证芯片标识符是否匹配
	se, err := h.seStorage.GetSeBySeId(session.Chips[partIndex-1])
	if err != nil {
		clog.Error("验证安全芯片标识符失败",
			clog.Err(err),
			clog.String("session_key", sessionKey),
			clog.String("se_id", session.Chips[partIndex-1]))

		// 更新会话状态为失败
		session.Status = model.StatusFailed

		// 通知发起者失败
		h.notifySessionFailure(sessionKey, sender,
			"验证安全芯片标识符失败",
			err.Error())

		return fmt.Errorf("验证安全芯片标识符失败: %w", err)
	}

	if se.CPIC != cpic {
		errMsg := fmt.Sprintf("安全芯片标识符不匹配: %s != %s", se.CPIC, cpic)
		clog.Error(errMsg,
			clog.String("session_key", sessionKey),
			clog.String("expected_cpic", se.CPIC),
			clog.String("actual_cpic", cpic))

		// 更新会话状态为失败
		session.Status = model.StatusFailed

		// 通知发起者失败
		h.notifySessionFailure(sessionKey, sender,
			fmt.Sprintf("参与方 %s 的安全芯片标识符不匹配", sender.GetUserName()),
			errMsg)

		return fmt.Errorf(errMsg)
	}

	// 接受邀请
	session.Responses[partIndex-1] = string(model.ParticipantAccepted)
	clog.Debug("参与方接受密钥生成邀请",
		clog.String("session_key", sessionKey),
		clog.String("participant", sender.GetUserName()),
		clog.Int("part_index", partIndex))

	// 检查是否所有参与方都已接受，统计 session.Responses 是否全为 Accepted
	acceptedCount := 0
	for _, status := range session.Responses {
		if status == string(model.ParticipantAccepted) {
			acceptedCount++
		}
	}

	clog.Debug("密钥生成参与方接受状态",
		clog.String("session_key", sessionKey),
		clog.Int("accepted_count", acceptedCount),
		clog.Int("total_count", len(session.Responses)))

	if acceptedCount == len(session.Responses) {
		clog.Info("所有参与方已接受密钥生成邀请，开始发送参数",
			clog.String("session_key", sessionKey),
			clog.Int("participants_count", len(session.Participants)))

		// 记录发送失败的参与者
		failedParticipants := []string{}

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
				clog.Warn("参与方不在线，无法发送参数",
					clog.String("participant", participant),
					clog.String("session_key", sessionKey))
				failedParticipants = append(failedParticipants, participant)
				continue
			}

			if err := client.SendMessage(paramsMsg); err != nil {
				clog.Error("向参与方发送参数失败",
					clog.Err(err),
					clog.String("participant", participant),
					clog.String("session_key", sessionKey))
				failedParticipants = append(failedParticipants, participant)
			} else {
				clog.Debug("向参与方发送参数成功",
					clog.String("participant", participant),
					clog.String("session_key", sessionKey),
					clog.Int("part_index", i+1))
			}
		}

		// 检查是否有发送参数失败的情况
		if len(failedParticipants) > 0 {
			clog.Error("部分参与者发送参数失败",
				clog.Any("failed_participants", failedParticipants),
				clog.String("session_key", sessionKey))

			// 更新会话状态为失败
			session.Status = model.StatusFailed

			// 通知发起者失败
			failedMsg := fmt.Sprintf("向以下参与者发送参数失败: %v", failedParticipants)
			h.notifySessionFailure(sessionKey, sender, "密钥生成过程失败", failedMsg)

			return fmt.Errorf("部分参与者发送参数失败: %v", failedParticipants)
		}

		// 更新会话状态为处理中
		session.Status = model.StatusProcessing
		clog.Info("密钥生成会话状态已更新为处理中",
			clog.String("session_key", sessionKey),
			clog.String("status", string(model.StatusProcessing)))
	}

	return nil
}

// handleKeyGenResult 处理密钥生成结果
func (h *KeyGenHandler) handleKeyGenResult(msg KeyGenResultMessage, sender *Client) error {
	// 获取会话
	sessionKey := msg.SessionKey
	session := h.sessionManager.GetKeyGenSession(sessionKey)
	if session == nil {
		errMsg := fmt.Sprintf("找不到对应的密钥生成会话: %s", sessionKey)
		clog.Error(errMsg)
		return fmt.Errorf(errMsg)
	}

	// 如果会话状态已经是失败，直接返回
	if session.Status == model.StatusFailed {
		clog.Warn("会话已处于失败状态，忽略结果",
			clog.String("session_key", sessionKey),
			clog.String("participant", sender.GetUserName()))
		return nil
	}

	// 直接从消息结构体获取字段
	success := msg.Success
	if !success {
		clog.Error("密钥生成失败",
			clog.String("session_key", sessionKey),
			clog.String("participant", sender.GetUserName()),
			clog.String("message", msg.Message))

		// 更新会话状态为失败
		session.Status = model.StatusFailed

		// 通知发起者失败
		h.notifySessionFailure(sessionKey, sender,
			fmt.Sprintf("参与方 %s 的密钥生成失败", sender.GetUserName()),
			msg.Message)

		return fmt.Errorf("密钥生成失败: %s", msg.Message)
	}

	partIndex := msg.PartIndex
	address := msg.Address
	cpic := msg.CPIC
	encryptedShard := msg.EncryptedShard

	clog.Info("收到密钥生成结果",
		clog.String("session_key", sessionKey),
		clog.String("participant", sender.GetUserName()),
		clog.Int("part_index", partIndex),
		clog.String("address", address))

	clog.Debug("密钥生成结果详情",
		clog.String("session_key", sessionKey),
		clog.String("participant", sender.GetUserName()),
		clog.Int("part_index", partIndex),
		clog.String("cpic", cpic),
		clog.Int("encrypted_shard_length", len(encryptedShard)))

	// 保存私钥分片
	if err := h.shareStorage.CreateEthereumKeyShard(sender.GetUserName(), address, cpic, encryptedShard, partIndex); err != nil {
		clog.Error("保存密钥分片失败",
			clog.Err(err),
			clog.String("session_key", sessionKey),
			clog.String("participant", sender.GetUserName()),
			clog.String("address", address))

		// 更新会话状态为失败
		session.Status = model.StatusFailed

		// 通知发起者失败
		h.notifySessionFailure(sessionKey, sender,
			fmt.Sprintf("保存参与方 %s 的密钥分片失败", sender.GetUserName()),
			err.Error())

		return fmt.Errorf("保存密钥分片失败: %w", err)
	}

	// 标记该部分已完成
	session.Responses[partIndex-1] = string(model.ParticipantCompleted)
	clog.Debug("更新参与方完成状态",
		clog.String("session_key", sessionKey),
		clog.String("participant", sender.GetUserName()),
		clog.Int("part_index", partIndex))

	// 检查是否所有参与方都已完成
	allCompleted := true
	completedCount := 0
	for _, status := range session.Responses {
		if status == string(model.ParticipantCompleted) {
			completedCount++
		}
		if status != string(model.ParticipantCompleted) {
			allCompleted = false
		}
	}

	clog.Debug("密钥生成完成状态",
		clog.String("session_key", sessionKey),
		clog.Int("completed_count", completedCount),
		clog.Int("total_count", len(session.Responses)),
		clog.Bool("all_completed", allCompleted))

	if allCompleted {
		// 更新会话状态为完成
		session.Status = model.StatusCompleted
		clog.Info("密钥生成会话已完成",
			clog.String("session_key", sessionKey),
			clog.String("address", address))

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
				clog.Error("向发起者发送确认消息失败",
					clog.Err(err),
					clog.String("initiator", initiator),
					clog.String("session_key", sessionKey))
			} else {
				clog.Debug("向发起者发送确认消息成功",
					clog.String("initiator", initiator),
					clog.String("session_key", sessionKey))
			}
		} else {
			clog.Warn("发起者不在线，无法发送确认消息",
				clog.String("initiator", initiator),
				clog.String("session_key", sessionKey))
		}
	}

	return nil
}
