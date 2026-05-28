package ws

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"offline-server/manager"
	"offline-server/storage"
	"offline-server/storage/model"
	mem_storage "offline-server/ws/storage"
)

// SignHandler 签名消息处理器。
type SignHandler struct {
	shareStorage      storage.IShareStorage
	seStorage         storage.ISeStorage
	offlineKeyStorage storage.IOfflineKeyStorage
	signStorage       storage.ISignStorage
	auditStorage      storage.IAuditStorage
	sessionManager    *mem_storage.SessionManager
	managerRuntime    manager.SessionRuntime
}

// NewSignHandler 创建签名消息处理器。
func NewSignHandler(
	shareStorage storage.IShareStorage,
	seStorage storage.ISeStorage,
	offlineKeyStorage storage.IOfflineKeyStorage,
	signStorage storage.ISignStorage,
	auditStorage storage.IAuditStorage,
	sessionManager *mem_storage.SessionManager,
	managerRuntime manager.SessionRuntime,
) *SignHandler {
	if managerRuntime == nil {
		managerRuntime = manager.NewSessionRuntimeFromEnv()
	}
	return &SignHandler{
		shareStorage:      shareStorage,
		seStorage:         seStorage,
		offlineKeyStorage: offlineKeyStorage,
		signStorage:       signStorage,
		auditStorage:      auditStorage,
		sessionManager:    sessionManager,
		managerRuntime:    managerRuntime,
	}
}

// ProcessMessage 处理签名相关消息。
func (h *SignHandler) ProcessMessage(msgType MessageType, rawMessage []byte, sender *Client) error {
	switch msgType {
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
		return fmt.Errorf("不支持的签名消息类型: %s", msgType)
	}
}

func (h *SignHandler) handleSignRequest(msg SignRequestMessage, sender *Client) error {
	if msg.SessionKey == "" || msg.MessageHash == "" || msg.Address == "" || len(msg.Participants) == 0 {
		return h.failSender(sender, "签名参数无效", "session_key、message_hash、address、participants 不能为空")
	}

	key, err := h.offlineKeyStorage.GetOfflineKeyByAddress(msg.Address)
	if err != nil {
		return h.failSender(sender, "找不到签名地址对应的离线密钥", err.Error())
	}
	offlineKeyID := msg.OfflineKeyID
	if offlineKeyID == "" {
		offlineKeyID = key.OfflineKeyID
	} else if offlineKeyID != key.OfflineKeyID {
		return h.failSender(sender, "离线密钥编号与地址不匹配", fmt.Sprintf("expected=%s actual=%s", key.OfflineKeyID, offlineKeyID))
	}
	if key.Status != model.OfflineKeyStatusActive {
		return h.failSender(sender, "离线密钥不可用", string(key.Status))
	}
	if len(msg.Participants) < key.RequiredSigners {
		return h.failSender(sender, "签名参与人数不足", fmt.Sprintf("required=%d actual=%d", key.RequiredSigners, len(msg.Participants)))
	}

	managerSession, err := h.managerRuntime.StartSession(msg.SessionKey)
	if err != nil {
		return h.failSender(sender, "启动 MPC manager 失败", err.Error())
	}

	shardsByUser := make(map[string]*model.KeyShard, len(msg.Participants))
	seIDs := make([]string, 0, len(msg.Participants))
	parties := make([]int, 0, len(msg.Participants))
	for _, participant := range msg.Participants {
		shard, err := h.shareStorage.GetKeyShardForParticipant(participant, msg.Address)
		if err != nil {
			_ = h.managerRuntime.StopSession(msg.SessionKey)
			return h.failSender(sender, "获取参与方分片失败", fmt.Sprintf("%s: %v", participant, err))
		}
		se, err := h.seStorage.GetSeByCPLC(shard.SeCPLC)
		if err != nil {
			_ = h.managerRuntime.StopSession(msg.SessionKey)
			return h.failSender(sender, "获取参与方安全芯片失败", fmt.Sprintf("%s: %v", participant, err))
		}
		if se.Status != model.SeStatusActive {
			_ = h.managerRuntime.StopSession(msg.SessionKey)
			return h.failSender(sender, "安全芯片不可用", fmt.Sprintf("%s status=%s", se.SeID, se.Status))
		}
		shardsByUser[participant] = shard
		seIDs = append(seIDs, se.SeID)
		parties = append(parties, shard.ShardIndex)
	}
	sort.Ints(parties)
	partiesStr := joinInts(parties)

	session := model.SignSession{
		SessionKey:    msg.SessionKey,
		TaskNo:        msg.TaskNo,
		OfflineKeyID:  offlineKeyID,
		TransactionNo: msg.TransactionNo,
		Initiator:     sender.GetUserName(),
		Address:       msg.Address,
		MessageHash:   msg.MessageHash,
		ManagerAddr:   managerSession.ManagerURL,
		Room:          managerSession.Room,
		Participants:  model.StringSlice(msg.Participants),
		Parties:       partiesStr,
		Responses:     makeResponseSlice(len(msg.Participants), model.ParticipantInit),
		SeIDs:         model.StringSlice(seIDs),
		Status:        model.StatusCreated,
	}

	if _, err := h.signStorage.CreateSession(session); err != nil {
		_ = h.managerRuntime.StopSession(msg.SessionKey)
		return h.failSender(sender, "保存签名会话失败", err.Error())
	}
	if _, err := h.sessionManager.CreateSignSession(session); err != nil {
		_ = h.managerRuntime.StopSession(msg.SessionKey)
		return h.failSender(sender, "创建签名会话失败", err.Error())
	}
	h.audit(sender, "sign_session_create", "sign_session", msg.SessionKey, "success", "")

	var offlineParticipants []string
	for i, participant := range msg.Participants {
		shard := shardsByUser[participant]
		inviteMsg := SignInviteMessage{
			BaseMessage:  BaseMessage{Type: MsgSignInvite},
			SessionKey:   msg.SessionKey,
			MessageHash:  msg.MessageHash,
			Address:      msg.Address,
			PartyIndex:   shard.ShardIndex,
			SeID:         seIDs[i],
			Participants: msg.Participants,
			Display:      msg.Display,
		}
		client, exists := sender.Hub().GetClient(participant)
		if !exists {
			offlineParticipants = append(offlineParticipants, participant)
			continue
		}
		if err := client.SendMessage(inviteMsg); err != nil {
			offlineParticipants = append(offlineParticipants, participant)
		}
	}
	if len(offlineParticipants) > 0 {
		h.markSignFailed(msg.SessionKey)
		return h.failSender(sender, "部分参与者不在线或邀请失败", fmt.Sprintf("%v", offlineParticipants))
	}

	sessionPtr := h.sessionManager.GetSignSession(msg.SessionKey)
	sessionPtr.Status = model.StatusInvited
	_ = h.signStorage.UpdateStatus(msg.SessionKey, model.StatusInvited)
	return nil
}

func (h *SignHandler) handleSignResponse(msg SignResponseMessage, sender *Client) error {
	session := h.sessionManager.GetSignSession(msg.SessionKey)
	if session == nil {
		return fmt.Errorf("找不到签名会话: %s", msg.SessionKey)
	}
	idx := indexOfParticipant(session.Participants, sender.GetUserName())
	if idx < 0 {
		return fmt.Errorf("参与者不属于会话: %s", sender.GetUserName())
	}

	if !msg.Accept {
		session.Responses[idx] = string(model.ParticipantRejected)
		h.markSignFailed(msg.SessionKey)
		h.notifySessionFailure(msg.SessionKey, sender, fmt.Sprintf("参与方 %s 拒绝签名", sender.GetUserName()), msg.Reason)
		return nil
	}

	shard, err := h.shareStorage.GetKeyShardForParticipant(sender.GetUserName(), session.Address)
	if err != nil {
		h.markSignFailed(msg.SessionKey)
		return fmt.Errorf("获取参与方分片失败: %w", err)
	}
	if shard.ShardIndex != msg.PartyIndex {
		h.markSignFailed(msg.SessionKey)
		return fmt.Errorf("party_index 不匹配: expected=%d actual=%d", shard.ShardIndex, msg.PartyIndex)
	}
	if shard.SeCPLC != msg.CPLC {
		h.markSignFailed(msg.SessionKey)
		return fmt.Errorf("CPLC 不匹配: expected=%s actual=%s", shard.SeCPLC, msg.CPLC)
	}

	session.Responses[idx] = string(model.ParticipantAccepted)
	_ = h.signStorage.UpdateParticipantStatus(msg.SessionKey, idx, model.ParticipantAccepted)
	if !allResponses(session.Responses, model.ParticipantAccepted) {
		return nil
	}

	parties, err := parseParties(session.Parties)
	if err != nil {
		h.markSignFailed(msg.SessionKey)
		return err
	}
	for _, participant := range session.Participants {
		shard, err := h.shareStorage.GetKeyShardForParticipant(participant, session.Address)
		if err != nil {
			h.markSignFailed(msg.SessionKey)
			return fmt.Errorf("获取参与方分片失败: %w", err)
		}
		signingIndex := indexOfInt(parties, shard.ShardIndex) + 1
		if signingIndex <= 0 {
			h.markSignFailed(msg.SessionKey)
			return fmt.Errorf("分片索引不在 parties 中: %d", shard.ShardIndex)
		}

		seSignature, err := SignData(shard.RecordID, session.Address)
		if err != nil {
			h.markSignFailed(msg.SessionKey)
			return fmt.Errorf("生成 SE 授权签名失败: %w", err)
		}
		paramsMsg := SignParamsMessage{
			BaseMessage:    BaseMessage{Type: MsgSignParams},
			SessionKey:     session.SessionKey,
			ManagerAddr:    session.ManagerAddr,
			Room:           session.Room,
			MessageHash:    session.MessageHash,
			Address:        session.Address,
			Signature:      seSignature,
			Parties:        session.Parties,
			PartyIndex:     shard.ShardIndex,
			SigningIndex:   signingIndex,
			RecordID:       shard.RecordID,
			FileName:       fmt.Sprintf("%s_sign_%d.json", session.SessionKey, signingIndex),
			EncryptedShard: shard.EncryptedBlob,
		}

		client, exists := sender.Hub().GetClient(participant)
		if !exists {
			h.markSignFailed(msg.SessionKey)
			return fmt.Errorf("参与者不在线，无法发送签名参数: %s", participant)
		}
		if err := client.SendMessage(paramsMsg); err != nil {
			h.markSignFailed(msg.SessionKey)
			return fmt.Errorf("发送签名参数失败: %w", err)
		}
	}

	session.Status = model.StatusProcessing
	_ = h.signStorage.UpdateStatus(msg.SessionKey, model.StatusProcessing)
	return nil
}

func (h *SignHandler) handleSignResult(msg SignResultMessage, sender *Client) error {
	session := h.sessionManager.GetSignSession(msg.SessionKey)
	if session == nil {
		return fmt.Errorf("找不到签名会话: %s", msg.SessionKey)
	}
	idx := indexOfParticipant(session.Participants, sender.GetUserName())
	if idx < 0 {
		return fmt.Errorf("参与者不属于会话: %s", sender.GetUserName())
	}
	if !msg.Success {
		h.markSignFailed(msg.SessionKey)
		h.notifySessionFailure(msg.SessionKey, sender, fmt.Sprintf("参与方 %s 签名失败", sender.GetUserName()), msg.Message)
		return nil
	}
	if msg.Signature == "" {
		h.markSignFailed(msg.SessionKey)
		return fmt.Errorf("签名结果为空")
	}
	if session.Signature != "" && session.Signature != msg.Signature {
		h.markSignFailed(msg.SessionKey)
		h.notifySessionFailure(msg.SessionKey, sender, "签名结果不一致", fmt.Sprintf("participant=%s", sender.GetUserName()))
		return fmt.Errorf("签名结果不一致")
	}

	session.Responses[idx] = string(model.ParticipantCompleted)
	session.Signature = msg.Signature
	_ = h.signStorage.UpdateParticipantStatus(msg.SessionKey, idx, model.ParticipantCompleted)

	if !allResponses(session.Responses, model.ParticipantCompleted) {
		return nil
	}

	session.Status = model.StatusCompleted
	_ = h.signStorage.UpdateSignature(msg.SessionKey, msg.Signature)
	_ = h.managerRuntime.StopSession(msg.SessionKey)
	h.audit(sender, "sign_session_complete", "sign_session", msg.SessionKey, "success", "")

	confirmMsg := SignCompleteMessage{
		BaseMessage: BaseMessage{Type: MsgSignComplete},
		SessionKey:  msg.SessionKey,
		Signature:   msg.Signature,
		Success:     true,
		Message:     "签名已完成",
	}
	if client, exists := sender.Hub().GetClient(session.Initiator); exists {
		_ = client.SendMessage(confirmMsg)
	}
	return nil
}

func (h *SignHandler) notifySessionFailure(sessionKey string, sender *Client, message string, details string) {
	session := h.sessionManager.GetSignSession(sessionKey)
	if session == nil {
		return
	}
	session.Status = model.StatusFailed
	failureMsg := ErrorMessage{BaseMessage: BaseMessage{Type: MsgError}, Message: message, Details: details}
	if client, exists := sender.Hub().GetClient(session.Initiator); exists {
		_ = client.SendMessage(failureMsg)
	}
}

func (h *SignHandler) failSender(sender *Client, message, details string) error {
	_ = sender.SendMessage(ErrorMessage{BaseMessage: BaseMessage{Type: MsgError}, Message: message, Details: details})
	return fmt.Errorf("%s: %s", message, details)
}

func (h *SignHandler) markSignFailed(sessionKey string) {
	if session := h.sessionManager.GetSignSession(sessionKey); session != nil {
		session.Status = model.StatusFailed
	}
	_ = h.signStorage.UpdateStatus(sessionKey, model.StatusFailed)
	if h.managerRuntime != nil {
		_ = h.managerRuntime.StopSession(sessionKey)
	}
}

func (h *SignHandler) audit(sender *Client, action, resourceType, resourceID, result, errMsg string) {
	if h.auditStorage == nil {
		return
	}
	_ = h.auditStorage.CreateAuditLog(model.AuditLog{
		Username:     sender.GetUserName(),
		Role:         string(sender.Role()),
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Result:       result,
		ErrorMessage: errMsg,
	})
}

func joinInts(values []int) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = strconv.Itoa(value)
	}
	return strings.Join(parts, ",")
}

func parseParties(parties string) ([]int, error) {
	if parties == "" {
		return nil, fmt.Errorf("parties 不能为空")
	}
	raw := strings.Split(parties, ",")
	values := make([]int, 0, len(raw))
	for _, item := range raw {
		value, err := strconv.Atoi(strings.TrimSpace(item))
		if err != nil {
			return nil, fmt.Errorf("parties 格式错误: %w", err)
		}
		values = append(values, value)
	}
	return values, nil
}

func indexOfInt(values []int, needle int) int {
	for i, value := range values {
		if value == needle {
			return i
		}
	}
	return -1
}

func indexOfParticipant(participants model.StringSlice, username string) int {
	for i, participant := range participants {
		if participant == username {
			return i
		}
	}
	return -1
}
