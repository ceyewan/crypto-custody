package ws

import (
	"encoding/json"
	"fmt"
	"time"

	"offline-server/storage"
	"offline-server/storage/model"
	mem_storage "offline-server/ws/storage"
)

// TransferHandler 处理分片移交双确认消息。
type TransferHandler struct {
	shareStorage   storage.IShareStorage
	auditStorage   storage.IAuditStorage
	approvalStore  storage.IApprovalStorage
	sessionManager *mem_storage.SessionManager
}

func NewTransferHandler(
	shareStorage storage.IShareStorage,
	auditStorage storage.IAuditStorage,
	approvalStore storage.IApprovalStorage,
	sessionManager *mem_storage.SessionManager,
) *TransferHandler {
	return &TransferHandler{
		shareStorage:   shareStorage,
		auditStorage:   auditStorage,
		approvalStore:  approvalStore,
		sessionManager: sessionManager,
	}
}

func (h *TransferHandler) ProcessMessage(msgType MessageType, rawMessage []byte, sender *Client) error {
	switch msgType {
	case MsgTransferRequest:
		var msg TransferRequestMessage
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			return fmt.Errorf("解析分片移交请求失败: %w", err)
		}
		return h.handleTransferRequest(msg, sender)
	case MsgTransferResponse:
		var msg TransferResponseMessage
		if err := json.Unmarshal(rawMessage, &msg); err != nil {
			return fmt.Errorf("解析分片移交响应失败: %w", err)
		}
		return h.handleTransferResponse(msg, sender)
	default:
		return fmt.Errorf("不支持的分片移交消息类型: %s", msgType)
	}
}

func (h *TransferHandler) handleTransferRequest(msg TransferRequestMessage, sender *Client) error {
	if sender.Role() != RoleAdmin {
		return h.failSender(sender, "权限不足", "只有管理员可以发起分片移交")
	}
	if msg.SessionKey == "" || msg.ShardID == "" || msg.FromUsername == "" || msg.ToUsername == "" {
		return h.failSender(sender, "分片移交参数无效", "session_key、shard_id、from_username、to_username 不能为空")
	}
	if msg.FromUsername == msg.ToUsername {
		return h.failSender(sender, "分片移交参数无效", "移出警员和接收警员不能相同")
	}

	shard, err := h.shareStorage.GetKeyShardByID(msg.ShardID)
	if err != nil {
		return h.failSender(sender, "分片不存在", err.Error())
	}
	if shard.Status != model.KeyShardStatusActive {
		return h.failSender(sender, "分片不可移交", string(shard.Status))
	}
	if shard.Username != msg.FromUsername {
		return h.failSender(sender, "分片持有人不匹配", fmt.Sprintf("expected=%s actual=%s", shard.Username, msg.FromUsername))
	}

	session, err := h.sessionManager.CreateTransferSession(mem_storage.TransferSession{
		SessionKey:   msg.SessionKey,
		ShardID:      msg.ShardID,
		OfflineKeyID: shard.OfflineKeyID,
		Initiator:    sender.GetUserName(),
		Address:      shard.Address,
		CaseNo:       msg.CaseNo,
		ShardIndex:   shard.ShardIndex,
		FromUsername: msg.FromUsername,
		ToUsername:   msg.ToUsername,
		Participants: model.StringSlice{msg.FromUsername, msg.ToUsername},
		Responses:    makeResponseSlice(2, model.ParticipantInit),
		Status:       model.StatusCreated,
		Reason:       msg.Reason,
	})
	if err != nil {
		return h.failSender(sender, "创建分片移交会话失败", err.Error())
	}

	for _, username := range session.Participants {
		client, exists := sender.Hub().GetClient(username)
		if !exists {
			h.markTransferFailed(session.SessionKey)
			return h.failSender(sender, "移交确认方不在线", string(username))
		}
		if err := client.SendMessage(TransferInviteMessage{
			BaseMessage:  BaseMessage{Type: MsgTransferInvite},
			SessionKey:   session.SessionKey,
			ShardID:      session.ShardID,
			OfflineKeyID: session.OfflineKeyID,
			Address:      session.Address,
			CaseNo:       session.CaseNo,
			ShardIndex:   session.ShardIndex,
			FromUsername: session.FromUsername,
			ToUsername:   session.ToUsername,
			Initiator:    session.Initiator,
			Reason:       session.Reason,
			Summary:      "分片移交确认",
		}); err != nil {
			h.markTransferFailed(session.SessionKey)
			return h.failSender(sender, "发送分片移交邀请失败", err.Error())
		}
	}

	session.Status = model.StatusInvited
	h.audit(sender, "transfer_session_create", "key_shard", msg.ShardID, "success", "")
	return nil
}

func (h *TransferHandler) handleTransferResponse(msg TransferResponseMessage, sender *Client) error {
	session := h.sessionManager.GetTransferSession(msg.SessionKey)
	if session == nil {
		return fmt.Errorf("找不到分片移交会话: %s", msg.SessionKey)
	}
	idx := indexOfParticipant(session.Participants, sender.GetUserName())
	if idx < 0 {
		return fmt.Errorf("参与者不属于分片移交会话: %s", sender.GetUserName())
	}
	if msg.ShardID != "" && msg.ShardID != session.ShardID {
		return fmt.Errorf("分片编号不匹配: expected=%s actual=%s", session.ShardID, msg.ShardID)
	}

	if !msg.Accept {
		session.Responses[idx] = string(model.ParticipantRejected)
		h.markTransferFailed(session.SessionKey)
		h.recordApproval(session, sender.GetUserName(), model.ApprovalRejected)
		h.notifyTransferComplete(session, sender, false, fmt.Sprintf("参与方 %s 拒绝分片移交", sender.GetUserName()))
		return nil
	}

	session.Responses[idx] = string(model.ParticipantAccepted)
	if !allResponses(session.Responses, model.ParticipantAccepted) {
		return nil
	}

	updated, err := h.shareStorage.TransferKeyShard(session.ShardID, session.ToUsername)
	if err != nil {
		h.markTransferFailed(session.SessionKey)
		h.notifyTransferComplete(session, sender, false, "分片移交失败: "+err.Error())
		return err
	}
	session.Status = model.StatusCompleted
	session.Address = updated.Address
	h.audit(sender, "transfer_session_complete", "key_shard", session.ShardID, "success", fmt.Sprintf("from=%s,to=%s", session.FromUsername, session.ToUsername))
	h.recordApproval(session, sender.GetUserName(), model.ApprovalApproved)
	h.notifyTransferComplete(session, sender, true, "分片移交已完成")
	return nil
}

func (h *TransferHandler) notifyTransferComplete(session *mem_storage.TransferSession, sender *Client, success bool, message string) {
	complete := TransferCompleteMessage{
		BaseMessage:  BaseMessage{Type: MsgTransferComplete},
		SessionKey:   session.SessionKey,
		ShardID:      session.ShardID,
		Address:      session.Address,
		FromUsername: session.FromUsername,
		ToUsername:   session.ToUsername,
		Success:      success,
		Message:      message,
	}
	for _, username := range []string{session.Initiator, session.FromUsername, session.ToUsername} {
		if client, exists := sender.Hub().GetClient(username); exists {
			_ = client.SendMessage(complete)
		}
	}
}

func (h *TransferHandler) markTransferFailed(sessionKey string) {
	if session := h.sessionManager.GetTransferSession(sessionKey); session != nil {
		session.Status = model.StatusFailed
	}
}

func (h *TransferHandler) failSender(sender *Client, message, details string) error {
	_ = sender.SendMessage(ErrorMessage{BaseMessage: BaseMessage{Type: MsgError}, Message: message, Details: details})
	return fmt.Errorf("%s: %s", message, details)
}

func (h *TransferHandler) audit(sender *Client, action, resourceType, resourceID, result, errMsg string) {
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

func (h *TransferHandler) recordApproval(session *mem_storage.TransferSession, approvedBy string, status model.ApprovalStatus) {
	if h.approvalStore == nil || session == nil {
		return
	}
	_, _ = h.approvalStore.CreateApproval(model.Approval{
		ApprovalID:  fmt.Sprintf("APPROVAL-%s-%d", sanitizeApprovalPart(session.ShardID), time.Now().UnixNano()),
		Operation:   "offline_shard_transfer",
		ResourceID:  session.ShardID,
		RequestedBy: session.Initiator,
		ApprovedBy:  approvedBy,
		Role:        "officer",
		Status:      status,
	})
}

func sanitizeApprovalPart(value string) string {
	if value == "" {
		return "resource"
	}
	var out []rune
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			out = append(out, r)
		} else {
			out = append(out, '_')
		}
	}
	if len(out) == 0 {
		return "resource"
	}
	return string(out)
}
