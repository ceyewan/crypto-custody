package ws

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"offline-server/storage"
	"offline-server/storage/model"
	mem_storage "offline-server/ws/storage"
)

// DestroyHandler 处理密钥销毁消息。
type DestroyHandler struct {
	shareStorage      storage.IShareStorage
	seStorage         storage.ISeStorage
	offlineKeyStorage storage.IOfflineKeyStorage
	auditStorage      storage.IAuditStorage
	approvalStore     storage.IApprovalStorage
	sessionManager    *mem_storage.SessionManager
	destroyMu         sync.Mutex
}

// NewDestroyHandler 创建密钥销毁消息处理器。
func NewDestroyHandler(
	shareStorage storage.IShareStorage,
	seStorage storage.ISeStorage,
	offlineKeyStorage storage.IOfflineKeyStorage,
	auditStorage storage.IAuditStorage,
	approvalStore storage.IApprovalStorage,
	sessionManager *mem_storage.SessionManager,
) *DestroyHandler {
	return &DestroyHandler{
		shareStorage:      shareStorage,
		seStorage:         seStorage,
		offlineKeyStorage: offlineKeyStorage,
		auditStorage:      auditStorage,
		approvalStore:     approvalStore,
		sessionManager:    sessionManager,
	}
}

// ProcessMessage 处理密钥销毁相关消息。
func (h *DestroyHandler) ProcessMessage(msgType MessageType, rawMessage []byte, sender *Client) error {
	switch msgType {
	case MsgDestroyRequest:
		var msg DestroyRequestMessage
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			return fmt.Errorf("解析密钥销毁请求消息失败: %w", err)
		}
		return h.handleDestroyRequest(msg, sender)
	case MsgDestroyResponse:
		var msg DestroyResponseMessage
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			return fmt.Errorf("解析密钥销毁响应消息失败: %w", err)
		}
		return h.handleDestroyResponse(msg, sender)
	case MsgDestroyResult:
		var msg DestroyResultMessage
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			return fmt.Errorf("解析密钥销毁结果消息失败: %w", err)
		}
		return h.handleDestroyResult(msg, sender)
	default:
		return fmt.Errorf("不支持的密钥销毁消息类型: %s", msgType)
	}
}

func (h *DestroyHandler) handleDestroyRequest(msg DestroyRequestMessage, sender *Client) error {
	if msg.SessionKey == "" || (msg.OfflineKeyID == "" && msg.Address == "") {
		return h.failSender(sender, "密钥销毁参数无效", "session_key 以及 offline_key_id/address 不能为空")
	}

	h.destroyMu.Lock()
	defer h.destroyMu.Unlock()

	key, err := h.loadDestroyKey(msg)
	if err != nil {
		return h.failSender(sender, "离线密钥不存在", err.Error())
	}
	if !isDestroyRequestStatus(key.Status) {
		return h.failSender(sender, "离线密钥不可销毁", string(key.Status))
	}

	activeShards, err := h.listDestroyableShards(key.Address)
	if err != nil {
		return h.failSender(sender, "查询密钥分片失败", err.Error())
	}
	if len(activeShards) == 0 {
		if err := h.offlineKeyStorage.UpdateOfflineKeyStatus(key.OfflineKeyID, model.OfflineKeyStatusDestroyed); err != nil {
			return h.failSender(sender, "更新离线密钥销毁状态失败", err.Error())
		}
		return h.failSender(sender, "没有剩余可销毁分片", "离线密钥已标记为 destroyed")
	}
	shards, err := selectDestroyShards(activeShards, msg.Participants)
	if err != nil {
		return h.failSender(sender, "密钥销毁参与者无效", err.Error())
	}
	if err := h.offlineKeyStorage.UpdateOfflineKeyStatus(key.OfflineKeyID, model.OfflineKeyStatusDestroying); err != nil {
		return h.failSender(sender, "更新离线密钥销毁状态失败", err.Error())
	}

	participants := make(model.StringSlice, 0, len(shards))
	for _, shard := range shards {
		participants = append(participants, shard.Username)
	}

	session, err := h.sessionManager.CreateDestroySession(mem_storage.DestroySession{
		SessionKey:   msg.SessionKey,
		OfflineKeyID: key.OfflineKeyID,
		Initiator:    sender.GetUserName(),
		Address:      key.Address,
		Participants: participants,
		Responses:    makeResponseSlice(len(participants), model.ParticipantInit),
		Shards:       shards,
		Status:       model.StatusCreated,
		Reason:       msg.Reason,
	})
	if err != nil {
		_ = h.offlineKeyStorage.UpdateOfflineKeyStatus(key.OfflineKeyID, model.OfflineKeyStatusDestroyFailed)
		return h.failSender(sender, "创建密钥销毁会话失败", err.Error())
	}
	h.audit(sender, "destroy_session_create", "offline_key", key.OfflineKeyID, "success", "")

	var failed []string
	for _, shard := range session.Shards {
		se, err := h.seStorage.GetSeByCPLC(shard.SeCPLC)
		if err != nil || se.Status != model.SeStatusActive {
			failed = append(failed, shard.Username)
			continue
		}
		inviteMsg := DestroyInviteMessage{
			BaseMessage:  BaseMessage{Type: MsgDestroyInvite},
			SessionKey:   session.SessionKey,
			OfflineKeyID: session.OfflineKeyID,
			CaseNo:       key.CaseNo,
			Initiator:    sender.GetUserName(),
			Address:      session.Address,
			PartyIndex:   shard.ShardIndex,
			SeID:         se.SeID,
			Summary:      "分片销毁确认",
			Reason:       session.Reason,
		}
		client, exists := sender.Hub().GetClient(shard.Username)
		if !exists {
			failed = append(failed, shard.Username)
			continue
		}
		if err := client.SendMessage(inviteMsg); err != nil {
			failed = append(failed, shard.Username)
		}
	}
	if len(failed) > 0 {
		h.markDestroyFailed(msg.SessionKey)
		return h.failSender(sender, "部分参与者不在线或安全芯片不可用", fmt.Sprintf("%v", failed))
	}

	session.Status = model.StatusInvited
	return nil
}

func (h *DestroyHandler) handleDestroyResponse(msg DestroyResponseMessage, sender *Client) error {
	session := h.sessionManager.GetDestroySession(msg.SessionKey)
	if session == nil {
		return fmt.Errorf("找不到密钥销毁会话: %s", msg.SessionKey)
	}
	if session.Status == model.StatusCompleted {
		return nil
	}
	idx := indexOfParticipant(session.Participants, sender.GetUserName())
	if idx < 0 {
		return fmt.Errorf("参与者不属于销毁会话: %s", sender.GetUserName())
	}

	if !msg.Accept {
		session.Responses[idx] = string(model.ParticipantRejected)
		h.markDestroyFailed(msg.SessionKey)
		h.recordApproval(session, sender.GetUserName(), model.ApprovalRejected)
		h.notifySessionFailure(session, sender, fmt.Sprintf("参与方 %s 拒绝销毁", sender.GetUserName()), msg.Reason)
		return nil
	}

	shard := session.Shards[idx]
	if shard.ShardIndex != msg.PartyIndex {
		h.markDestroyFailed(msg.SessionKey)
		return fmt.Errorf("party_index 不匹配: expected=%d actual=%d", shard.ShardIndex, msg.PartyIndex)
	}
	if shard.SeCPLC != msg.CPLC {
		h.markDestroyFailed(msg.SessionKey)
		return fmt.Errorf("CPLC 不匹配: expected=%s actual=%s", shard.SeCPLC, msg.CPLC)
	}
	_ = h.seStorage.TouchSeLastUsedByCPLC(msg.CPLC)

	session.Responses[idx] = string(model.ParticipantAccepted)
	h.audit(sender, "destroy_participant_accept", "offline_key", session.OfflineKeyID, "success", fmt.Sprintf("party_index=%d,address=%s", msg.PartyIndex, session.Address))
	if !allResponses(session.Responses, model.ParticipantAccepted) {
		return nil
	}

	for _, shard := range session.Shards {
		if err := h.shareStorage.UpdateKeyShardStatus(shard.ShardID, model.KeyShardStatusDestroying); err != nil {
			h.markDestroyFailed(msg.SessionKey)
			return fmt.Errorf("标记分片销毁中失败: %w", err)
		}
	}

	for _, shard := range session.Shards {
		signature, err := SignData(shard.RecordID, session.Address)
		if err != nil {
			h.restoreDestroyingShards(session)
			h.markDestroyFailed(msg.SessionKey)
			return fmt.Errorf("生成 SE 删除授权签名失败: %w", err)
		}
		paramsMsg := DestroyParamsMessage{
			BaseMessage:  BaseMessage{Type: MsgDestroyParams},
			SessionKey:   session.SessionKey,
			OfflineKeyID: session.OfflineKeyID,
			Address:      session.Address,
			PartyIndex:   shard.ShardIndex,
			RecordID:     shard.RecordID,
			Signature:    signature,
		}
		client, exists := sender.Hub().GetClient(shard.Username)
		if !exists {
			h.restoreDestroyingShards(session)
			h.markDestroyFailed(msg.SessionKey)
			return fmt.Errorf("参与者不在线，无法发送销毁参数: %s", shard.Username)
		}
		if err := client.SendMessage(paramsMsg); err != nil {
			h.restoreDestroyingShards(session)
			h.markDestroyFailed(msg.SessionKey)
			return fmt.Errorf("发送销毁参数失败: %w", err)
		}
	}

	session.Status = model.StatusProcessing
	return nil
}

func (h *DestroyHandler) handleDestroyResult(msg DestroyResultMessage, sender *Client) error {
	session := h.sessionManager.GetDestroySession(msg.SessionKey)
	if session == nil {
		return fmt.Errorf("找不到密钥销毁会话: %s", msg.SessionKey)
	}
	if session.Status == model.StatusCompleted {
		return nil
	}
	idx := indexOfParticipant(session.Participants, sender.GetUserName())
	if idx < 0 {
		return fmt.Errorf("参与者不属于销毁会话: %s", sender.GetUserName())
	}
	shard := session.Shards[idx]
	if shard.ShardIndex != msg.PartyIndex {
		return fmt.Errorf("party_index 不匹配: expected=%d actual=%d", shard.ShardIndex, msg.PartyIndex)
	}
	if !msg.Success {
		if session.Responses[idx] == string(model.ParticipantCompleted) {
			return nil
		}
		session.Responses[idx] = string(model.ParticipantFailed)
		_ = h.shareStorage.UpdateKeyShardStatus(shard.ShardID, model.KeyShardStatusActive)
		h.markDestroyFailed(msg.SessionKey)
		h.recordApproval(session, sender.GetUserName(), model.ApprovalRejected)
		h.notifySessionFailure(session, sender, fmt.Sprintf("参与方 %s 销毁失败", sender.GetUserName()), msg.Message)
		return nil
	}

	if session.Responses[idx] == string(model.ParticipantCompleted) {
		return h.completeDestroyIfNoActiveShards(session, sender)
	}
	session.Responses[idx] = string(model.ParticipantCompleted)
	if err := h.shareStorage.UpdateKeyShardStatus(shard.ShardID, model.KeyShardStatusDestroyed); err != nil {
		h.markDestroyFailed(msg.SessionKey)
		return fmt.Errorf("更新分片销毁状态失败: %w", err)
	}
	_ = h.seStorage.TouchSeLastUsedByCPLC(shard.SeCPLC)
	h.audit(sender, "destroy_participant_complete", "offline_key", session.OfflineKeyID, "success", fmt.Sprintf("party_index=%d,address=%s", msg.PartyIndex, session.Address))

	if !allResponses(session.Responses, model.ParticipantCompleted) {
		return h.completeDestroyIfNoActiveShards(session, sender)
	}

	return h.completeDestroyIfNoActiveShards(session, sender)
}

func (h *DestroyHandler) completeDestroyIfNoActiveShards(session *mem_storage.DestroySession, sender *Client) error {
	shards, err := h.shareStorage.ListKeyShardsByAddress(session.Address)
	if err != nil {
		h.markDestroyFailed(session.SessionKey)
		return fmt.Errorf("查询剩余分片失败: %w", err)
	}
	remaining := 0
	for _, shard := range shards {
		if shard.Status == model.KeyShardStatusActive || shard.Status == model.KeyShardStatusDestroying {
			remaining++
		}
	}
	if remaining > 0 {
		if allResponses(session.Responses, model.ParticipantCompleted) {
			h.markDestroyFailed(session.SessionKey)
			h.notifySessionFailure(session, sender, "密钥销毁未完成", fmt.Sprintf("remaining_live_shards=%d", remaining))
		}
		return nil
	}

	if err := h.offlineKeyStorage.UpdateOfflineKeyStatus(session.OfflineKeyID, model.OfflineKeyStatusDestroyed); err != nil {
		h.markDestroyFailed(session.SessionKey)
		return fmt.Errorf("更新离线密钥销毁状态失败: %w", err)
	}
	session.Status = model.StatusCompleted
	h.audit(sender, "destroy_session_complete", "offline_key", session.OfflineKeyID, "success", "")
	h.recordApproval(session, sender.GetUserName(), model.ApprovalApproved)

	completeMsg := DestroyCompleteMessage{
		BaseMessage:  BaseMessage{Type: MsgDestroyComplete},
		SessionKey:   session.SessionKey,
		OfflineKeyID: session.OfflineKeyID,
		Address:      session.Address,
		Destroyed:    len(session.Shards),
		Success:      true,
		Message:      "密钥销毁已完成",
	}
	if client, exists := sender.Hub().GetClient(session.Initiator); exists {
		_ = client.SendMessage(completeMsg)
	}
	return nil
}

func (h *DestroyHandler) loadDestroyKey(msg DestroyRequestMessage) (*model.OfflineKey, error) {
	if msg.OfflineKeyID != "" {
		key, err := h.offlineKeyStorage.GetOfflineKeyByID(msg.OfflineKeyID)
		if err != nil {
			return nil, err
		}
		if msg.Address != "" && key.Address != msg.Address {
			return nil, fmt.Errorf("离线密钥编号与地址不匹配: expected=%s actual=%s", key.Address, msg.Address)
		}
		return key, nil
	}
	return h.offlineKeyStorage.GetOfflineKeyByAddress(msg.Address)
}

func (h *DestroyHandler) listDestroyableShards(address string) ([]model.KeyShard, error) {
	shards, err := h.shareStorage.ListKeyShardsByAddress(address)
	if err != nil {
		return nil, err
	}
	var out []model.KeyShard
	for _, shard := range shards {
		if shard.Status == model.KeyShardStatusActive || shard.Status == model.KeyShardStatusDestroying {
			out = append(out, shard)
		}
	}
	return out, nil
}

func (h *DestroyHandler) restoreDestroyingShards(session *mem_storage.DestroySession) {
	if session == nil {
		return
	}
	for _, shard := range session.Shards {
		_ = h.shareStorage.UpdateKeyShardStatus(shard.ShardID, model.KeyShardStatusActive)
	}
}

func selectDestroyShards(activeShards []model.KeyShard, participants []string) ([]model.KeyShard, error) {
	if len(activeShards) == 0 {
		return nil, fmt.Errorf("没有可销毁的 active 分片")
	}
	if len(participants) == 0 {
		shards := append([]model.KeyShard(nil), activeShards...)
		sort.Slice(shards, func(i, j int) bool {
			return shards[i].ShardIndex < shards[j].ShardIndex
		})
		return shards, nil
	}

	byUser := make(map[string]model.KeyShard, len(activeShards))
	for _, shard := range activeShards {
		byUser[shard.Username] = shard
	}
	shards := make([]model.KeyShard, 0, len(participants))
	seen := map[string]struct{}{}
	for _, participant := range participants {
		if _, ok := seen[participant]; ok {
			return nil, fmt.Errorf("参与者重复: %s", participant)
		}
		shard, ok := byUser[participant]
		if !ok {
			return nil, fmt.Errorf("参与者没有 active 分片: %s", participant)
		}
		seen[participant] = struct{}{}
		shards = append(shards, shard)
	}
	return shards, nil
}

func isDestroyRequestStatus(status model.OfflineKeyStatus) bool {
	return status == model.OfflineKeyStatusActive || status == model.OfflineKeyStatusDestroyFailed
}

func (h *DestroyHandler) notifySessionFailure(session *mem_storage.DestroySession, sender *Client, message string, details string) {
	session.Status = model.StatusFailed
	failureMsg := ErrorMessage{BaseMessage: BaseMessage{Type: MsgError}, Message: message, Details: details}
	if client, exists := sender.Hub().GetClient(session.Initiator); exists {
		_ = client.SendMessage(failureMsg)
	}
}

func (h *DestroyHandler) failSender(sender *Client, message, details string) error {
	_ = sender.SendMessage(ErrorMessage{BaseMessage: BaseMessage{Type: MsgError}, Message: message, Details: details})
	return fmt.Errorf("%s: %s", message, details)
}

func (h *DestroyHandler) markDestroyFailed(sessionKey string) {
	if session := h.sessionManager.GetDestroySession(sessionKey); session != nil {
		session.Status = model.StatusFailed
		if session.OfflineKeyID != "" {
			_ = h.offlineKeyStorage.UpdateOfflineKeyStatus(session.OfflineKeyID, model.OfflineKeyStatusDestroyFailed)
		}
	}
}

func (h *DestroyHandler) audit(sender *Client, action, resourceType, resourceID, result, errMsg string) {
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

func (h *DestroyHandler) recordApproval(session *mem_storage.DestroySession, approvedBy string, status model.ApprovalStatus) {
	if h.approvalStore == nil || session == nil {
		return
	}
	_, _ = h.approvalStore.CreateApproval(model.Approval{
		ApprovalID:  fmt.Sprintf("APPROVAL-%s-%d", sanitizeApprovalPart(session.OfflineKeyID), time.Now().UnixNano()),
		Operation:   "offline_key_destroy",
		ResourceID:  session.OfflineKeyID,
		RequestedBy: session.Initiator,
		ApprovedBy:  approvedBy,
		Role:        "officer",
		Status:      status,
	})
}
