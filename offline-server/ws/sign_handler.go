package ws

import (
	"encoding/json"
	"fmt"
	"strings"

	"offline-server/clog"
	"offline-server/storage"
	"offline-server/storage/model"
	mem_storage "offline-server/ws/storage"
)

// SignHandler 签名消息处理器
type SignHandler struct {
	shareStorage   storage.IShareStorage       // 私钥分片存储接口
	seStorage      storage.ISeStorage          // 安全芯片存储接口
	sessionManager *mem_storage.SessionManager // 会话管理器
}

// NewSignHandler 创建签名消息处理器
func NewSignHandler(shareStorage storage.IShareStorage, seStorage storage.ISeStorage, sessionManager *mem_storage.SessionManager) *SignHandler {
	handler := &SignHandler{
		shareStorage:   shareStorage,
		seStorage:      seStorage,
		sessionManager: sessionManager,
	}

	clog.Debug("创建签名消息处理器实例")
	return handler
}

// ProcessMessage 处理签名相关消息
func (h *SignHandler) ProcessMessage(msgType MessageType, rawMessage []byte, sender *Client) error {
	clog.Debug("处理签名消息",
		clog.String("msg_type", string(msgType)),
		clog.String("username", sender.GetUserName()),
		clog.Int("msg_size", len(rawMessage)))

	// 根据消息类型分发处理
	switch msgType {
	case MsgSignRequest:
		var msg SignRequestMessage
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			clog.Error("解析签名请求消息失败",
				clog.Err(err),
				clog.String("username", sender.GetUserName()))
			return fmt.Errorf("解析签名请求消息失败: %w", err)
		}
		return h.handleSignRequest(msg, sender)

	case MsgSignResponse:
		var msg SignResponseMessage
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			clog.Error("解析签名响应消息失败",
				clog.Err(err),
				clog.String("username", sender.GetUserName()))
			return fmt.Errorf("解析签名响应消息失败: %w", err)
		}
		return h.handleSignResponse(msg, sender)

	case MsgSignResult:
		var msg SignResultMessage
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			clog.Error("解析签名结果消息失败",
				clog.Err(err),
				clog.String("username", sender.GetUserName()))
			return fmt.Errorf("解析签名结果消息失败: %w", err)
		}
		return h.handleSignResult(msg, sender)

	default:
		clog.Error("不支持的签名消息类型",
			clog.String("msg_type", string(msgType)),
			clog.String("username", sender.GetUserName()))
		return fmt.Errorf("不支持的签名消息类型: %s", msgType)
	}
}

// notifySessionFailure 通知签名会话发起者失败
func (h *SignHandler) notifySessionFailure(sessionKey string, sender *Client, message string, details string) {
	session := h.sessionManager.GetSignSession(sessionKey)
	if session == nil {
		clog.Warn("无法获取签名会话信息，无法发送失败通知",
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

// handleSignRequest 处理签名请求
func (h *SignHandler) handleSignRequest(msg SignRequestMessage, sender *Client) error {
	// 直接从消息结构体获取字段
	sessionKey := msg.SessionKey
	data := msg.Data
	address := msg.Address
	participants := msg.Participants

	clog.Info("收到签名请求",
		clog.String("session_key", sessionKey),
		clog.String("initiator", sender.GetUserName()),
		clog.String("address", address),
		clog.Int("participants_count", len(participants)))

	clog.Debug("签名请求详情",
		clog.String("session_key", sessionKey),
		clog.String("data", data),
		clog.String("address", address),
		clog.Any("participants", participants))

	// 创建签名会话
	session, err := h.sessionManager.CreateSignSession(sessionKey, sender.GetUserName(), data, address, participants)
	if err != nil {
		clog.Error("创建签名会话失败",
			clog.Err(err),
			clog.String("session_key", sessionKey))

		// 直接向发起者发送失败消息
		failureMsg := ErrorMessage{
			BaseMessage: BaseMessage{Type: MsgError},
			Message:     "创建签名会话失败",
			Details:     err.Error(),
		}
		if sendErr := sender.SendMessage(failureMsg); sendErr != nil {
			clog.Error("向发起者发送失败通知失败",
				clog.Err(sendErr),
				clog.String("username", sender.GetUserName()))
		}

		return fmt.Errorf("创建签名会话失败: %w", err)
	}

	// 为每个参与者分配安全芯片ID
	chips, err := h.seStorage.GetRandomSeIds(len(participants))
	if err != nil {
		clog.Error("获取安全芯片标识符失败",
			clog.Err(err),
			clog.Int("requested_count", len(participants)))

		// 更新会话状态为失败
		session.Status = model.StatusFailed

		// 通知发起者失败
		h.notifySessionFailure(sessionKey, sender, "获取安全芯片标识符失败", err.Error())

		return fmt.Errorf("获取安全芯片标识符失败: %w", err)
	}
	session.Chips = chips

	clog.Debug("获取到安全芯片标识符",
		clog.Int("chip_count", len(chips)),
		clog.Any("chips", chips))

	// 记录离线参与者
	offlineParticipants := []string{}

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
		} else {
			clog.Debug("向参与方发送邀请成功",
				clog.String("participant", participant),
				clog.String("session_key", sessionKey),
				clog.Int("part_index", i+1))
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
		h.notifySessionFailure(sessionKey, sender, "签名初始化失败", offlineMsg)

		return fmt.Errorf("部分参与者离线或发送邀请失败: %v", offlineParticipants)
	}

	// 更新会话状态为已邀请
	session.Status = model.StatusInvited
	clog.Info("签名会话状态已更新为已邀请",
		clog.String("session_key", sessionKey),
		clog.String("status", string(model.StatusInvited)))

	return nil
}

// handleSignResponse 处理签名响应
func (h *SignHandler) handleSignResponse(msg SignResponseMessage, sender *Client) error {
	// 直接从消息结构体获取字段
	sessionKey := msg.SessionKey
	partIndex := msg.PartIndex
	cpic := msg.CPIC
	accept := msg.Accept
	reason := msg.Reason

	// 获取会话
	session := h.sessionManager.GetSignSession(sessionKey)
	if session == nil {
		errMsg := fmt.Sprintf("找不到对应的签名会话: %s", sessionKey)
		clog.Error(errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// 如果会话状态已经是失败，直接返回
	if session.Status == model.StatusFailed {
		clog.Warn("会话已处于失败状态，忽略响应",
			clog.String("session_key", sessionKey),
			clog.String("participant", sender.GetUserName()))
		return nil
	}

	clog.Info("收到签名响应",
		clog.String("session_key", sessionKey),
		clog.String("participant", sender.GetUserName()),
		clog.Int("part_index", partIndex),
		clog.Bool("accept", accept))

	if !accept {
		// 拒绝邀请
		session.Responses[partIndex-1] = string(model.ParticipantRejected)
		clog.Info("参与方拒绝签名邀请",
			clog.String("session_key", sessionKey),
			clog.String("participant", sender.GetUserName()),
			clog.Int("part_index", partIndex),
			clog.String("reason", reason))

		// 更新会话状态为失败
		session.Status = model.StatusFailed

		// 通知发起者有参与方拒绝
		h.notifySessionFailure(sessionKey, sender,
			fmt.Sprintf("参与方 %s 拒绝了签名邀请", sender.GetUserName()),
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

		return fmt.Errorf("%s", errMsg)
	}

	// 接受邀请
	session.Responses[partIndex-1] = string(model.ParticipantAccepted)
	clog.Debug("参与方接受签名邀请",
		clog.String("session_key", sessionKey),
		clog.String("participant", sender.GetUserName()),
		clog.Int("part_index", partIndex))

	// 检查是否所有参与方都已接受
	allAccepted := true
	acceptedCount := 0
	for _, status := range session.Responses {
		if status == string(model.ParticipantAccepted) {
			acceptedCount++
		}
		if status != string(model.ParticipantAccepted) {
			allAccepted = false
		}
	}

	clog.Debug("签名参与方接受状态",
		clog.String("session_key", sessionKey),
		clog.Int("accepted_count", acceptedCount),
		clog.Int("total_count", len(session.Responses)),
		clog.Bool("all_accepted", allAccepted))

	if allAccepted {
		clog.Info("所有参与方已接受签名邀请，开始发送参数",
			clog.String("session_key", sessionKey),
			clog.Int("participants_count", len(session.Participants)))

		// 记录发送失败的参与者
		failedParticipants := []string{}

		// 向所有参与方发送参数
		for _, participant := range session.Participants {
			// 获取该参与者的密钥分片数据
			encryptedShard, err := h.shareStorage.GetEthereumKeyShard(participant, session.Address)
			if err != nil {
				clog.Error("获取参与方的密钥分片失败",
					clog.Err(err),
					clog.String("participant", participant),
					clog.String("address", session.Address))
				failedParticipants = append(failedParticipants, participant)
				continue
			}

			clog.Debug("获取到参与方密钥分片",
				clog.String("participant", participant),
				clog.String("address", session.Address),
				clog.Int("shard_index", encryptedShard.ShardIndex))

			// 生成签名参与者列表(索引集合)
			parties := make([]int, len(session.Participants))
			for j := range session.Participants {
				parties[j] = j + 1
			}
			// 将参与者列表转换为字符串，例如 [1,2,3] => "1,2,3"
			partiesStr := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(parties)), ","), "[]")

			// 获取签名，用于读取安全芯片
			sign, err := SignData(participant, session.Address)
			if err != nil {
				clog.Error("获取签名失败",
					clog.Err(err),
					clog.String("participant", participant),
					clog.String("address", session.Address))
				failedParticipants = append(failedParticipants, participant)
				continue
			}

			// 准备参数消息
			paramsMsg := SignParamsMessage{
				BaseMessage:    BaseMessage{Type: MsgSignParams},
				SessionKey:     sessionKey,
				Data:           session.Data,
				Address:        session.Address,
				Signature:      sign,
				Parties:        partiesStr,
				PartIndex:      encryptedShard.ShardIndex,
				FileName:       fmt.Sprintf("%s_%d.json", sessionKey, encryptedShard.ShardIndex),
				EncryptedShard: encryptedShard.PrivateShard,
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
					clog.Int("shard_index", encryptedShard.ShardIndex))
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
			h.notifySessionFailure(sessionKey, sender, "签名过程失败", failedMsg)

			return fmt.Errorf("部分参与者发送参数失败: %v", failedParticipants)
		}

		// 更新会话状态为处理中
		session.Status = model.StatusProcessing
		clog.Info("签名会话状态已更新为处理中",
			clog.String("session_key", sessionKey),
			clog.String("status", string(model.StatusProcessing)))
	}

	return nil
}

// handleSignResult 处理签名结果
func (h *SignHandler) handleSignResult(msg SignResultMessage, sender *Client) error {
	// 直接从消息结构体获取字段
	sessionKey := msg.SessionKey
	partIndex := msg.PartIndex
	signature := msg.Signature
	success := msg.Success
	message := msg.Message

	// 获取会话
	session := h.sessionManager.GetSignSession(sessionKey)
	if session == nil {
		errMsg := fmt.Sprintf("找不到对应的签名会话: %s", sessionKey)
		clog.Error(errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// 如果会话状态已经是失败，直接返回
	if session.Status == model.StatusFailed {
		clog.Warn("会话已处于失败状态，忽略结果",
			clog.String("session_key", sessionKey),
			clog.String("participant", sender.GetUserName()))
		return nil
	}

	clog.Info("收到签名结果",
		clog.String("session_key", sessionKey),
		clog.String("participant", sender.GetUserName()),
		clog.Int("part_index", partIndex),
		clog.Bool("success", success))

	// 检查签名是否成功
	if !success {
		clog.Error("签名失败",
			clog.String("session_key", sessionKey),
			clog.String("participant", sender.GetUserName()),
			clog.String("message", message))

		// 更新会话状态为失败
		session.Status = model.StatusFailed

		// 通知发起者失败
		h.notifySessionFailure(sessionKey, sender,
			fmt.Sprintf("参与方 %s 的签名失败", sender.GetUserName()),
			message)

		return fmt.Errorf("签名失败: %s", message)
	}

	clog.Debug("签名结果详情",
		clog.String("session_key", sessionKey),
		clog.String("participant", sender.GetUserName()),
		clog.Int("part_index", partIndex),
		clog.Int("signature_length", len(signature)))

	// 更新会话状态
	session.Responses[partIndex-1] = string(model.ParticipantCompleted)
	session.Signature = signature
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

	clog.Debug("签名完成状态",
		clog.String("session_key", sessionKey),
		clog.Int("completed_count", completedCount),
		clog.Int("total_count", len(session.Responses)),
		clog.Bool("all_completed", allCompleted))

	if allCompleted {
		// 更新会话状态为完成
		session.Status = model.StatusCompleted
		clog.Info("签名已完成",
			clog.String("session_key", sessionKey),
			clog.String("address", session.Address),
			clog.Int("signature_length", len(signature)))

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
				clog.Error("向发起者发送完成消息失败",
					clog.Err(err),
					clog.String("initiator", initiator),
					clog.String("session_key", sessionKey))
			} else {
				clog.Debug("向发起者发送完成消息成功",
					clog.String("initiator", initiator),
					clog.String("session_key", sessionKey))
			}
		} else {
			clog.Warn("发起者不在线，无法发送完成消息",
				clog.String("initiator", initiator),
				clog.String("session_key", sessionKey))
		}
	}

	return nil
}
