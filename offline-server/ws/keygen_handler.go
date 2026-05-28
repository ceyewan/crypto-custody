package ws

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"offline-server/manager"
	"offline-server/storage"
	"offline-server/storage/model"
	mem_storage "offline-server/ws/storage"
)

// KeyGenHandler 密钥生成消息处理器。
type KeyGenHandler struct {
	shareStorage      storage.IShareStorage
	seStorage         storage.ISeStorage
	offlineKeyStorage storage.IOfflineKeyStorage
	keyGenStorage     storage.IKeyGenStorage
	auditStorage      storage.IAuditStorage
	sessionManager    *mem_storage.SessionManager
	managerRuntime    manager.SessionRuntime
}

// NewKeyGenHandler 创建密钥生成消息处理器。
func NewKeyGenHandler(
	shareStorage storage.IShareStorage,
	seStorage storage.ISeStorage,
	offlineKeyStorage storage.IOfflineKeyStorage,
	keyGenStorage storage.IKeyGenStorage,
	auditStorage storage.IAuditStorage,
	sessionManager *mem_storage.SessionManager,
	managerRuntime manager.SessionRuntime,
) *KeyGenHandler {
	if managerRuntime == nil {
		managerRuntime = manager.NewSessionRuntimeFromEnv()
	}
	return &KeyGenHandler{
		shareStorage:      shareStorage,
		seStorage:         seStorage,
		offlineKeyStorage: offlineKeyStorage,
		keyGenStorage:     keyGenStorage,
		auditStorage:      auditStorage,
		sessionManager:    sessionManager,
		managerRuntime:    managerRuntime,
	}
}

// ProcessMessage 处理密钥生成相关消息。
func (h *KeyGenHandler) ProcessMessage(msgType MessageType, rawMessage []byte, sender *Client) error {
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
	default:
		return fmt.Errorf("不支持的密钥生成消息类型: %s", msgType)
	}
}

func (h *KeyGenHandler) handleKeyGenRequest(msg KeyGenRequestMessage, sender *Client) error {
	if msg.SessionKey == "" || msg.RequiredSigners <= 0 || msg.TotalParties <= 0 ||
		msg.RequiredSigners > msg.TotalParties || len(msg.Participants) != msg.TotalParties {
		return h.failSender(sender, "密钥生成参数无效", "required_signers、total_parties、participants 不匹配")
	}

	offlineKeyID := msg.OfflineKeyID
	if offlineKeyID == "" {
		offlineKeyID = "key_" + msg.SessionKey
	}
	coinType := msg.CoinType
	if coinType == "" {
		coinType = "ETH"
	}
	managerSession, err := h.managerRuntime.StartSession(msg.SessionKey)
	if err != nil {
		return h.failSender(sender, "启动 MPC manager 失败", err.Error())
	}

	seIDs, err := h.seStorage.GetActiveSeIds(msg.TotalParties)
	if err != nil {
		_ = h.managerRuntime.StopSession(msg.SessionKey)
		return h.failSender(sender, "获取可用安全芯片失败", err.Error())
	}

	session := model.KeyGenSession{
		SessionKey:      msg.SessionKey,
		TaskNo:          msg.TaskNo,
		CaseNo:          msg.CaseNo,
		OfflineKeyID:    offlineKeyID,
		CoinType:        coinType,
		Initiator:       sender.GetUserName(),
		RequiredSigners: msg.RequiredSigners,
		TotalParties:    msg.TotalParties,
		GG20Threshold:   msg.RequiredSigners - 1,
		ManagerAddr:     managerSession.ManagerURL,
		Room:            managerSession.Room,
		Participants:    model.StringSlice(msg.Participants),
		Responses:       makeResponseSlice(len(msg.Participants), model.ParticipantInit),
		SeIDs:           model.StringSlice(seIDs),
		Status:          model.StatusCreated,
	}

	if err := h.sessionManager.CreateKeyGenSession(session); err != nil {
		_ = h.managerRuntime.StopSession(msg.SessionKey)
		return h.failSender(sender, "创建密钥生成会话失败", err.Error())
	}
	if _, err := h.keyGenStorage.CreateSession(session); err != nil {
		_ = h.managerRuntime.StopSession(msg.SessionKey)
		return h.failSender(sender, "保存密钥生成会话失败", err.Error())
	}
	h.audit(sender, "keygen_session_create", "keygen_session", msg.SessionKey, "success", "")

	var offlineParticipants []string
	for i, participant := range msg.Participants {
		inviteMsg := KeyGenInviteMessage{
			BaseMessage:     BaseMessage{Type: MsgKeyGenInvite},
			SessionKey:      msg.SessionKey,
			TaskNo:          msg.TaskNo,
			CaseNo:          msg.CaseNo,
			Initiator:       sender.GetUserName(),
			CoinType:        coinType,
			RequiredSigners: msg.RequiredSigners,
			TotalParties:    msg.TotalParties,
			PartyIndex:      i + 1,
			SeID:            seIDs[i],
			Participants:    msg.Participants,
			Summary:         "密钥生成邀请",
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
		h.markKeyGenFailed(msg.SessionKey)
		return h.failSender(sender, "部分参与者不在线或邀请失败", fmt.Sprintf("%v", offlineParticipants))
	}

	sessionPtr := h.sessionManager.GetKeyGenSession(msg.SessionKey)
	sessionPtr.Status = model.StatusInvited
	_ = h.keyGenStorage.UpdateStatus(msg.SessionKey, model.StatusInvited)
	return nil
}

func (h *KeyGenHandler) handleKeyGenResponse(msg KeyGenResponseMessage, sender *Client) error {
	session := h.sessionManager.GetKeyGenSession(msg.SessionKey)
	if session == nil {
		return fmt.Errorf("找不到密钥生成会话: %s", msg.SessionKey)
	}
	idx := msg.PartyIndex - 1
	if idx < 0 || idx >= len(session.Participants) {
		return fmt.Errorf("party_index 无效: %d", msg.PartyIndex)
	}

	if !msg.Accept {
		session.Responses[idx] = string(model.ParticipantRejected)
		h.markKeyGenFailed(msg.SessionKey)
		h.notifySessionFailure(msg.SessionKey, sender, fmt.Sprintf("参与方 %s 拒绝密钥生成", sender.GetUserName()), msg.Reason)
		return nil
	}

	se, err := h.seStorage.GetSeBySeId(session.SeIDs[idx])
	if err != nil {
		h.markKeyGenFailed(msg.SessionKey)
		return fmt.Errorf("获取安全芯片失败: %w", err)
	}
	if se.CPLC != msg.CPLC || se.Status != model.SeStatusActive {
		h.markKeyGenFailed(msg.SessionKey)
		return fmt.Errorf("安全芯片不匹配或不可用: expected=%s actual=%s status=%s", se.CPLC, msg.CPLC, se.Status)
	}

	session.Responses[idx] = string(model.ParticipantAccepted)
	_ = h.keyGenStorage.UpdateParticipantStatus(msg.SessionKey, idx, model.ParticipantAccepted)

	if !allResponses(session.Responses, model.ParticipantAccepted) {
		return nil
	}

	for i, participant := range session.Participants {
		recordID := deriveRecordID(session.OfflineKeyID, i+1, 1)
		paramsMsg := KeyGenParamsMessage{
			BaseMessage:  BaseMessage{Type: MsgKeyGenParams},
			SessionKey:   session.SessionKey,
			ManagerAddr:  session.ManagerAddr,
			Room:         session.Room,
			Threshold:    session.GG20Threshold,
			TotalParties: session.TotalParties,
			PartyIndex:   i + 1,
			RecordID:     recordID,
			FileName:     fmt.Sprintf("%s_keygen_%d.json", session.SessionKey, i+1),
		}

		client, exists := sender.Hub().GetClient(participant)
		if !exists {
			h.markKeyGenFailed(msg.SessionKey)
			return fmt.Errorf("参与者不在线，无法发送 keygen 参数: %s", participant)
		}
		if err := client.SendMessage(paramsMsg); err != nil {
			h.markKeyGenFailed(msg.SessionKey)
			return fmt.Errorf("发送 keygen 参数失败: %w", err)
		}
	}

	session.Status = model.StatusProcessing
	_ = h.keyGenStorage.UpdateStatus(msg.SessionKey, model.StatusProcessing)
	return nil
}

func (h *KeyGenHandler) handleKeyGenResult(msg KeyGenResultMessage, sender *Client) error {
	session := h.sessionManager.GetKeyGenSession(msg.SessionKey)
	if session == nil {
		return fmt.Errorf("找不到密钥生成会话: %s", msg.SessionKey)
	}
	idx := msg.PartyIndex - 1
	if idx < 0 || idx >= len(session.Participants) {
		return fmt.Errorf("party_index 无效: %d", msg.PartyIndex)
	}
	if !msg.Success {
		h.markKeyGenFailed(msg.SessionKey)
		h.notifySessionFailure(msg.SessionKey, sender, fmt.Sprintf("参与方 %s keygen 失败", sender.GetUserName()), msg.Message)
		return nil
	}
	if session.AccountAddr != "" && session.AccountAddr != msg.Address {
		h.markKeyGenFailed(msg.SessionKey)
		return fmt.Errorf("keygen 地址不一致: expected=%s actual=%s", session.AccountAddr, msg.Address)
	}

	recordID := msg.RecordID
	if recordID == "" {
		recordID = deriveRecordID(session.OfflineKeyID, msg.PartyIndex, 1)
	}
	expectedSe, err := h.seStorage.GetSeBySeId(session.SeIDs[idx])
	if err != nil {
		h.markKeyGenFailed(msg.SessionKey)
		return fmt.Errorf("获取分配安全芯片失败: %w", err)
	}
	if expectedSe.CPLC != msg.CPLC {
		h.markKeyGenFailed(msg.SessionKey)
		return fmt.Errorf("keygen 结果 CPLC 不匹配: expected=%s actual=%s", expectedSe.CPLC, msg.CPLC)
	}
	if _, err := h.seStorage.GetSeByCPLC(msg.CPLC); err != nil {
		h.markKeyGenFailed(msg.SessionKey)
		return fmt.Errorf("结果中的 CPLC 未登记: %w", err)
	}

	shard := model.KeyShard{
		ShardID:       fmt.Sprintf("%s:%d", session.OfflineKeyID, msg.PartyIndex),
		OfflineKeyID:  session.OfflineKeyID,
		Username:      sender.GetUserName(),
		Address:       msg.Address,
		ShardIndex:    msg.PartyIndex,
		RecordID:      recordID,
		SeCPLC:        msg.CPLC,
		EncryptedBlob: msg.EncryptedShard,
		BlobType:      model.BlobTypeMPCShare,
		KeyVersion:    1,
		Status:        model.KeyShardStatusActive,
	}
	if _, err := h.shareStorage.CreateKeyShard(shard); err != nil {
		h.markKeyGenFailed(msg.SessionKey)
		return fmt.Errorf("保存密钥分片失败: %w", err)
	}

	if session.AccountAddr == "" {
		session.AccountAddr = msg.Address
		session.PublicKey = msg.PublicKey
		_ = h.keyGenStorage.UpdateAccountAddr(msg.SessionKey, msg.Address)
	}
	session.Responses[idx] = string(model.ParticipantCompleted)
	_ = h.keyGenStorage.UpdateParticipantStatus(msg.SessionKey, idx, model.ParticipantCompleted)

	if !allResponses(session.Responses, model.ParticipantCompleted) {
		return nil
	}

	if _, err := h.offlineKeyStorage.CreateOfflineKey(model.OfflineKey{
		OfflineKeyID:    session.OfflineKeyID,
		TaskNo:          session.TaskNo,
		CaseNo:          session.CaseNo,
		Address:         session.AccountAddr,
		CoinType:        session.CoinType,
		Algorithm:       model.AlgorithmGG20ECDSASECP256K1,
		RequiredSigners: session.RequiredSigners,
		TotalParties:    session.TotalParties,
		PublicKey:       session.PublicKey,
		LogicalOwner:    session.Initiator,
		Status:          model.OfflineKeyStatusActive,
	}); err != nil {
		h.markKeyGenFailed(msg.SessionKey)
		return fmt.Errorf("保存离线密钥元数据失败: %w", err)
	}

	session.Status = model.StatusCompleted
	_ = h.keyGenStorage.UpdateStatus(msg.SessionKey, model.StatusCompleted)
	_ = h.managerRuntime.StopSession(msg.SessionKey)
	h.audit(sender, "keygen_session_complete", "offline_key", session.OfflineKeyID, "success", "")

	confirmMsg := KeyGenCompleteMessage{
		BaseMessage: BaseMessage{Type: MsgKeyGenComplete},
		SessionKey:  msg.SessionKey,
		Address:     session.AccountAddr,
		Success:     true,
		Message:     "密钥生成已完成",
	}
	if client, exists := sender.Hub().GetClient(session.Initiator); exists {
		_ = client.SendMessage(confirmMsg)
	}
	return nil
}

func (h *KeyGenHandler) notifySessionFailure(sessionKey string, sender *Client, message string, details string) {
	session := h.sessionManager.GetKeyGenSession(sessionKey)
	if session == nil {
		return
	}
	session.Status = model.StatusFailed
	failureMsg := ErrorMessage{BaseMessage: BaseMessage{Type: MsgError}, Message: message, Details: details}
	if client, exists := sender.Hub().GetClient(session.Initiator); exists {
		_ = client.SendMessage(failureMsg)
	}
}

func (h *KeyGenHandler) failSender(sender *Client, message, details string) error {
	_ = sender.SendMessage(ErrorMessage{BaseMessage: BaseMessage{Type: MsgError}, Message: message, Details: details})
	return fmt.Errorf("%s: %s", message, details)
}

func (h *KeyGenHandler) markKeyGenFailed(sessionKey string) {
	if session := h.sessionManager.GetKeyGenSession(sessionKey); session != nil {
		session.Status = model.StatusFailed
	}
	_ = h.keyGenStorage.UpdateStatus(sessionKey, model.StatusFailed)
	if h.managerRuntime != nil {
		_ = h.managerRuntime.StopSession(sessionKey)
	}
}

func (h *KeyGenHandler) audit(sender *Client, action, resourceType, resourceID, result, errMsg string) {
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

func deriveRecordID(offlineKeyID string, shardIndex int, keyVersion int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("offline-secret:v1|%s|%d|%d", offlineKeyID, shardIndex, keyVersion)))
	return hex.EncodeToString(sum[:])
}

func makeResponseSlice(count int, status model.ParticipantStatus) model.StringSlice {
	responses := make(model.StringSlice, count)
	for i := range responses {
		responses[i] = string(status)
	}
	return responses
}

func allResponses(responses model.StringSlice, status model.ParticipantStatus) bool {
	if len(responses) == 0 {
		return false
	}
	for _, current := range responses {
		if current != string(status) {
			return false
		}
	}
	return true
}
