package handler

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"offline-server/storage"
	storagedb "offline-server/storage/db"
	"offline-server/storage/model"

	"github.com/gin-gonic/gin"
)

var (
	offlineTaskStorage = storage.GetOfflineTaskStorage()
	offlineKeyStorage  = storage.GetOfflineKeyStorage()
	offlineShareStore  = storage.GetShareStorage()
	offlineSignStore   = storage.GetSignStorage()
	offlineAuditStore  = storage.GetAuditStorage()
	offlineApproval    = storage.GetApprovalStorage()
)

const offlineDBPath = "data/crypto-custody.db"

type offlinePackage struct {
	SchemaVersion    string          `json:"schema_version"`
	PackageType      string          `json:"package_type"`
	TaskType         string          `json:"task_type"`
	TaskNo           string          `json:"task_no"`
	SourceSystem     string          `json:"source_system"`
	TargetSystem     string          `json:"target_system"`
	CreatedBy        string          `json:"created_by"`
	CreatedAt        string          `json:"created_at"`
	Payload          json.RawMessage `json:"payload"`
	PayloadHash      string          `json:"payload_hash"`
	PackageSignature any             `json:"package_signature"`
}

type offlineTaskResponse struct {
	ID             uint                    `json:"id"`
	TaskNo         string                  `json:"task_no"`
	TaskType       string                  `json:"task_type"`
	SourceSystem   string                  `json:"source_system"`
	PayloadHash    string                  `json:"payload_hash"`
	ResultHash     string                  `json:"result_hash"`
	RawPackagePath string                  `json:"raw_package_path"`
	Status         model.OfflineTaskStatus `json:"status"`
	CreatedAt      time.Time               `json:"created_at"`
	UpdatedAt      time.Time               `json:"updated_at"`
}

type offlineKeyResponse struct {
	ID              uint                   `json:"id"`
	OfflineKeyID    string                 `json:"offline_key_id"`
	TaskNo          string                 `json:"task_no"`
	CaseNo          string                 `json:"case_no"`
	Address         string                 `json:"address"`
	CoinType        string                 `json:"coin_type"`
	Algorithm       model.Algorithm        `json:"algorithm"`
	RequiredSigners int                    `json:"required_signers"`
	TotalParties    int                    `json:"total_parties"`
	PublicKey       string                 `json:"public_key"`
	LogicalOwner    string                 `json:"logical_owner"`
	Status          model.OfflineKeyStatus `json:"status"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Shards          []keyShardResponse     `json:"shards,omitempty"`
}

type keyShardResponse struct {
	ShardID             string               `json:"shard_id"`
	OfflineKeyID        string               `json:"offline_key_id"`
	Address             string               `json:"address"`
	CaseNo              string               `json:"case_no,omitempty"`
	TaskNo              string               `json:"task_no,omitempty"`
	RequiredSigners     int                  `json:"required_signers,omitempty"`
	TotalParties        int                  `json:"total_parties,omitempty"`
	Username            string               `json:"username"`
	ShardIndex          int                  `json:"shard_index"`
	RecordID            string               `json:"record_id"`
	SeCPLC              string               `json:"se_cplc"`
	BlobType            model.BlobType       `json:"blob_type"`
	KeyVersion          int                  `json:"key_version"`
	Status              model.KeyShardStatus `json:"status"`
	EncryptedBlobSize   int                  `json:"encrypted_blob_size"`
	EncryptedBlobSHA256 string               `json:"encrypted_blob_sha256"`
	CreatedAt           time.Time            `json:"created_at"`
	UpdatedAt           time.Time            `json:"updated_at"`
}

type auditLogResponse struct {
	ID                uint      `json:"id"`
	CreatedAt         time.Time `json:"created_at"`
	Username          string    `json:"username"`
	Role              string    `json:"role"`
	Action            string    `json:"action"`
	ResourceType      string    `json:"resource_type"`
	ResourceID        string    `json:"resource_id"`
	Result            string    `json:"result"`
	ErrorMessage      string    `json:"error_message,omitempty"`
	RedactedDetail    string    `json:"redacted_detail,omitempty"`
	SensitiveRedacted bool      `json:"sensitive_redacted"`
}

type approvalResponse struct {
	ID          uint                 `json:"id"`
	CreatedAt   time.Time            `json:"created_at"`
	ApprovalID  string               `json:"approval_id"`
	Operation   string               `json:"operation"`
	ResourceID  string               `json:"resource_id"`
	RequestedBy string               `json:"requested_by"`
	ApprovedBy  string               `json:"approved_by"`
	Role        string               `json:"role"`
	Status      model.ApprovalStatus `json:"status"`
}

type coldBackupRequest struct {
	Password string `json:"password"`
}

type restoreBackupRequest struct {
	Password string `json:"password"`
}

type encryptedBackupFile struct {
	Version    int    `json:"version"`
	KDF        string `json:"kdf"`
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
	PlainHash  string `json:"plainHash"`
}

type participationResponse struct {
	CreatedAt      time.Time `json:"created_at"`
	Type           string    `json:"type"`
	Action         string    `json:"action"`
	ResourceType   string    `json:"resource_type"`
	ResourceID     string    `json:"resource_id"`
	Result         string    `json:"result"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	RedactedDetail string    `json:"redacted_detail,omitempty"`
}

// ImportOfflineTask 导入在线系统导出的离线任务包。
func ImportOfflineTask(c *gin.Context) {
	raw, err := readPackageBody(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var pkg offlinePackage
	if err := json.Unmarshal(raw, &pkg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务包JSON格式错误: " + err.Error()})
		return
	}
	if err := validateOfflineTaskPackage(pkg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payloadHash, err := hashPayload(pkg.Payload)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	legacyPayloadHash, _ := hashPayloadCompact(pkg.Payload)
	if pkg.PayloadHash != payloadHash && pkg.PayloadHash != legacyPayloadHash {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload_hash不匹配"})
		return
	}

	if existing, err := offlineTaskStorage.GetTask(pkg.TaskNo); err == nil {
		if existing.PayloadHash != payloadHash && existing.PayloadHash != legacyPayloadHash && existing.PayloadHash != pkg.PayloadHash {
			c.JSON(http.StatusConflict, gin.H{"error": "任务编号已存在但payload_hash不同"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "task": taskDTO(existing), "payload": payloadMap(pkg.Payload), "duplicated": true})
		return
	}

	path, err := saveRawPackage(pkg.TaskNo, raw)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	task, err := offlineTaskStorage.CreateTask(model.OfflineTask{
		TaskNo:         pkg.TaskNo,
		TaskType:       pkg.TaskType,
		SourceSystem:   pkg.SourceSystem,
		PayloadHash:    payloadHash,
		RawPackagePath: path,
		Status:         model.OfflineTaskImported,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存离线任务失败: " + err.Error()})
		return
	}
	auditFromContext(c, "offline_task_import", "offline_task", pkg.TaskNo, "success", "")
	c.JSON(http.StatusOK, gin.H{"success": true, "task": taskDTO(task), "payload": payloadMap(pkg.Payload)})
}

// GetOfflineTask 查询导入任务摘要。
func GetOfflineTask(c *gin.Context) {
	taskNo := c.Param("task_no")
	task, _, payload, err := loadTaskPackage(taskNo)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "task": taskDTO(task), "payload": payload})
}

// ListOfflineTasks 查询已导入任务列表。
func ListOfflineTasks(c *gin.Context) {
	tasks, err := offlineTaskStorage.ListTasks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询离线任务列表失败"})
		return
	}
	responses := make([]offlineTaskResponse, 0, len(tasks))
	for _, task := range tasks {
		taskCopy := task
		responses = append(responses, taskDTO(&taskCopy))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "tasks": responses})
}

// BuildKeygenTaskRequest 返回基于任务包的 keygen_request 模板。
func BuildKeygenTaskRequest(c *gin.Context) {
	task, pkg, payload, err := loadTaskPackage(c.Param("task_no"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if task.TaskType != "custody_keygen" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务类型不是 custody_keygen"})
		return
	}

	var req struct {
		SessionKey   string   `json:"session_key"`
		OfflineKeyID string   `json:"offline_key_id"`
		Participants []string `json:"participants"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.SessionKey == "" {
		req.SessionKey = fmt.Sprintf("keygen_%s_%s", time.Now().UTC().Format("20060102150405"), task.TaskNo)
	}
	if req.OfflineKeyID == "" {
		req.OfflineKeyID = "OFFKEY-" + task.TaskNo
	}

	policy, _ := payload["threshold_policy"].(map[string]any)
	requiredSigners := intFromPayload(policy["required_signers"])
	totalParties := intFromPayload(policy["total_parties"])
	if requiredSigners <= 0 || totalParties <= 0 || requiredSigners > totalParties {
		c.JSON(http.StatusBadRequest, gin.H{"error": "threshold_policy无效"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": gin.H{
			"type":             "keygen_request",
			"session_key":      req.SessionKey,
			"task_no":          pkg.TaskNo,
			"case_no":          stringFromPayload(payload["case_no"]),
			"offline_key_id":   req.OfflineKeyID,
			"coin_type":        firstNonEmpty(stringFromPayload(payload["coin_type"]), "ETH"),
			"required_signers": requiredSigners,
			"total_parties":    totalParties,
			"participants":     req.Participants,
		},
	})
}

// BuildSignTaskRequest 返回基于任务包的 sign_request 模板。
func BuildSignTaskRequest(c *gin.Context) {
	task, pkg, payload, err := loadTaskPackage(c.Param("task_no"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	if task.TaskType != "sign" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "任务类型不是 sign"})
		return
	}

	var req struct {
		SessionKey   string   `json:"session_key"`
		OfflineKeyID string   `json:"offline_key_id"`
		Participants []string `json:"participants"`
	}
	_ = c.ShouldBindJSON(&req)
	if req.SessionKey == "" {
		req.SessionKey = fmt.Sprintf("sign_%s_%s", time.Now().UTC().Format("20060102150405"), task.TaskNo)
	}
	messageHash := stringFromPayload(payload["message_hash"])
	address := stringFromPayload(payload["from_address"])
	if messageHash == "" || address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "签名任务缺少message_hash或from_address"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": gin.H{
			"type":           "sign_request",
			"session_key":    req.SessionKey,
			"task_no":        pkg.TaskNo,
			"case_no":        stringFromPayload(payload["case_no"]),
			"offline_key_id": req.OfflineKeyID,
			"transaction_no": stringFromPayload(payload["transaction_no"]),
			"message_hash":   messageHash,
			"address":        address,
			"participants":   req.Participants,
			"display":        payload["display"],
		},
	})
}

// DownloadOfflineResult 导出离线结果包。
func DownloadOfflineResult(c *gin.Context) {
	task, pkg, payload, err := loadTaskPackage(c.Param("task_no"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	resultTaskType := ""
	resultPayload := map[string]any{}
	switch task.TaskType {
	case "custody_keygen":
		key, err := offlineKeyStorage.GetOfflineKeyByTaskNo(task.TaskNo)
		if err != nil || key.Status != model.OfflineKeyStatusActive {
			c.JSON(http.StatusConflict, gin.H{"error": "keygen结果尚未完成或密钥不可用"})
			return
		}
		resultTaskType = "custody_keygen_result"
		resultPayload = map[string]any{
			"case_no":         payload["case_no"],
			"coin_type":       key.CoinType,
			"chain_id":        payload["chain_id"],
			"custody_address": key.Address,
			"public_key":      key.PublicKey,
			"threshold_policy": map[string]any{
				"required_signers": key.RequiredSigners,
				"total_parties":    key.TotalParties,
			},
			"offline_ref_no": key.OfflineKeyID,
			"completed_at":   time.Now().UTC().Format(time.RFC3339),
		}
	case "sign":
		session, err := offlineSignStore.GetSessionByTaskNo(task.TaskNo)
		if err != nil || session.Status != model.StatusCompleted || session.Signature == "" {
			c.JSON(http.StatusConflict, gin.H{"error": "签名结果尚未完成"})
			return
		}
		resultTaskType = "sign_result"
		resultPayload = map[string]any{
			"case_no":          payload["case_no"],
			"transaction_no":   payload["transaction_no"],
			"coin_type":        payload["coin_type"],
			"chain_id":         payload["chain_id"],
			"from_address":     payload["from_address"],
			"message_hash":     session.MessageHash,
			"signature":        session.Signature,
			"signature_format": "ethereum_rsv",
			"offline_ref_no":   "OFFLINE-SIGN-" + task.TaskNo,
			"completed_at":     time.Now().UTC().Format(time.RFC3339),
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的任务类型"})
		return
	}

	resultPkg, resultHash, err := buildResultPackage(pkg, resultTaskType, resultPayload, usernameFromContext(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = offlineTaskStorage.UpdateTaskResultHash(task.TaskNo, resultHash)
	_ = offlineTaskStorage.UpdateTaskStatus(task.TaskNo, model.OfflineTaskCompleted)
	auditFromContext(c, "offline_result_download", "offline_task", task.TaskNo, "success", "")

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=offline_result_%s.json", sanitizeFilePart(task.TaskNo)))
	c.JSON(http.StatusOK, resultPkg)
}

// GetOfflineKey 查询离线密钥元数据。
func GetOfflineKey(c *gin.Context) {
	id := c.Param("offline_key_id")
	key, err := offlineKeyStorage.GetOfflineKeyByID(id)
	if err != nil {
		key, err = offlineKeyStorage.GetOfflineKeyByAddress(id)
	}
	if err != nil {
		auditFromContext(c, "offline_key_query", "offline_key", id, "failure", "not_found")
		c.JSON(http.StatusNotFound, gin.H{"error": "离线密钥不存在"})
		return
	}
	shards, _ := offlineShareStore.ListKeyShardsByAddress(key.Address)
	auditFromContext(c, "offline_key_query", "offline_key", key.OfflineKeyID, "success", fmt.Sprintf("shards=%d", len(shards)))
	c.JSON(http.StatusOK, gin.H{"success": true, "key": offlineKeyDTO(key, shards)})
}

// ListOfflineKeys 查询离线密钥列表。
func ListOfflineKeys(c *gin.Context) {
	keys, err := offlineKeyStorage.ListOfflineKeys()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询离线密钥列表失败"})
		return
	}
	responses := make([]offlineKeyResponse, 0, len(keys))
	for _, key := range keys {
		responses = append(responses, offlineKeyDTO(&key, nil))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "keys": responses})
}

// ListMyKeyShards 查询当前用户持有的分片。
func ListMyKeyShards(c *gin.Context) {
	username := usernameFromContext(c)
	shards, err := offlineShareStore.ListKeyShardsByUsername(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询我的分片失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "shards": shardDTOsWithKeyInfo(shards)})
}

// ListMyParticipationRecords 查询当前用户自己的参与记录。
func ListMyParticipationRecords(c *gin.Context) {
	username := usernameFromContext(c)
	logs, err := offlineAuditStore.ListAuditLogs(intFromQuery(c, "limit", 200))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询参与记录失败"})
		return
	}

	records := make([]participationResponse, 0)
	for _, log := range logs {
		if log.Username != username {
			continue
		}
		records = append(records, participationResponse{
			CreatedAt:      log.CreatedAt,
			Type:           participationType(log.Action),
			Action:         log.Action,
			ResourceType:   log.ResourceType,
			ResourceID:     log.ResourceID,
			Result:         log.Result,
			ErrorMessage:   log.ErrorMessage,
			RedactedDetail: log.RedactedDetail,
		})
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "records": records})
}

// ListKeyShards 管理员查询全部分片。
func ListKeyShards(c *gin.Context) {
	shards, err := offlineShareStore.ListKeyShards()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询分片列表失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "shards": shardDTOsWithKeyInfo(filterShards(shards, c))})
}

// TransferKeyShard 生成单个分片移交 WebSocket 请求。真正更新由双方确认后的 transfer_response 完成。
func TransferKeyShard(c *gin.Context) {
	shardID := c.Param("shard_id")
	var req struct {
		SessionKey  string `json:"session_key"`
		NewUsername string `json:"new_username"`
		ToUsername  string `json:"to_username"`
		Reason      string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
		return
	}
	targetUsername := firstNonEmpty(req.NewUsername, req.ToUsername)
	if strings.TrimSpace(targetUsername) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "接收警员不能为空"})
		return
	}

	shard, err := offlineShareStore.GetKeyShardByID(shardID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "分片不存在"})
		return
	}
	if shard.Status != model.KeyShardStatusActive {
		c.JSON(http.StatusConflict, gin.H{"error": "只能移交 active 分片"})
		return
	}
	if shard.Username == targetUsername {
		c.JSON(http.StatusBadRequest, gin.H{"error": "接收警员不能与当前持有人相同"})
		return
	}
	user, err := storage.GetUserStorage().GetUserByUsername(targetUsername)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "接收警员不存在"})
		return
	}
	if user.Role != model.RoleAdmin && user.Role != model.RoleOfficer {
		c.JSON(http.StatusBadRequest, gin.H{"error": "接收方必须是管理员或警员"})
		return
	}
	if req.SessionKey == "" {
		req.SessionKey = fmt.Sprintf("transfer_%s_%s", time.Now().UTC().Format("20060102150405"), sanitizeFilePart(shardID))
	}

	fromUsername := shard.Username
	auditFromContext(c, "offline_shard_transfer_request", "key_shard", shardID, "success", fmt.Sprintf("from=%s,to=%s,reason=%s", fromUsername, targetUsername, req.Reason))

	shardInfo := shardDTOWithKeyInfo(*shard)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": gin.H{
			"type":           "transfer_request",
			"session_key":    req.SessionKey,
			"shard_id":       shard.ShardID,
			"offline_key_id": shard.OfflineKeyID,
			"address":        shard.Address,
			"case_no":        shardInfo.CaseNo,
			"shard_index":    shard.ShardIndex,
			"from_username":  fromUsername,
			"to_username":    targetUsername,
			"reason":         req.Reason,
		},
	})
}

// DestroyOfflineKey 生成密钥销毁 WebSocket 请求。真正的状态变更由 destroy_result 完成。
func DestroyOfflineKey(c *gin.Context) {
	id := c.Param("offline_key_id")
	var req struct {
		SessionKey   string   `json:"session_key"`
		Participants []string `json:"participants"`
		Reason       string   `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)

	key, err := offlineKeyStorage.GetOfflineKeyByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "离线密钥不存在"})
		return
	}
	if !isDestroyableOfflineKeyStatus(key.Status) {
		c.JSON(http.StatusConflict, gin.H{"error": "离线密钥当前状态不可销毁"})
		return
	}
	activeShards, err := offlineShareStore.ListActiveKeyShardsByAddress(key.Address)
	if err != nil || len(activeShards) == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "没有可销毁的active分片"})
		return
	}
	if req.SessionKey == "" {
		req.SessionKey = fmt.Sprintf("destroy_%s_%s", time.Now().UTC().Format("20060102150405"), key.OfflineKeyID)
	}
	if len(req.Participants) == 0 {
		for _, shard := range activeShards {
			req.Participants = append(req.Participants, shard.Username)
		}
	}

	auditFromContext(c, "offline_key_destroy_request", "offline_key", id, "success", req.Reason)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": gin.H{
			"type":           "destroy_request",
			"session_key":    req.SessionKey,
			"offline_key_id": key.OfflineKeyID,
			"address":        key.Address,
			"participants":   req.Participants,
			"reason":         req.Reason,
		},
	})
}

func isDestroyableOfflineKeyStatus(status model.OfflineKeyStatus) bool {
	return status == model.OfflineKeyStatusActive || status == model.OfflineKeyStatusDestroyFailed
}

// ListAuditLogs 查询脱敏审计日志。
func ListAuditLogs(c *gin.Context) {
	filter := auditFilterFromQuery(c)
	page, pageSize := filter.Page, filter.PageSize
	logs, total, err := offlineAuditStore.SearchAuditLogs(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询审计日志失败"})
		return
	}
	items := auditLogDTOs(logs)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"logs":    items,
		"data": gin.H{
			"items":    items,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}

// ListApprovals 查询敏感操作审批记录。
func ListApprovals(c *gin.Context) {
	page := intFromQuery(c, "page", 1)
	pageSize := firstIntFromQuery(c, []string{"pageSize", "page_size", "limit"}, 20)
	approvals, total, err := offlineApproval.ListApprovalsPage(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询审批记录失败"})
		return
	}
	items := approvalDTOs(approvals)
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"approvals": items,
		"data": gin.H{
			"items":    items,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		},
	})
}

// DownloadBackup 下载当前 SQLite 数据库快照。
func DownloadBackup(c *gin.Context) {
	record, err := createOfflineHotBackupRecord(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "热备份创建失败: " + err.Error()})
		return
	}
	auditFromContext(c, "offline_backup_download", "backup", record.BackupNo, "success", "")
	c.FileAttachment(record.FilePath, record.FileName)
}

func CreateHotBackup(c *gin.Context) {
	record, err := createOfflineHotBackupRecord(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "热备份创建失败: " + err.Error()})
		return
	}
	auditFromContext(c, "offline_backup_hot_create", "backup", record.BackupNo, "success", "")
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "热备份创建成功", "data": record})
}

func CreateColdBackup(c *gin.Context) {
	var req coldBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}
	record, err := createOfflineColdBackupRecord(c, req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "冷备份创建失败: " + err.Error()})
		return
	}
	auditFromContext(c, "offline_backup_cold_create", "backup", record.BackupNo, "success", "")
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "冷备份创建成功", "data": record})
}

func ListBackups(c *gin.Context) {
	var backups []model.BackupRecord
	database := storagedb.GetDB()
	if err := database.Order("created_at DESC").Find(&backups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询备份记录失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "查询备份记录成功", "data": backups})
}

func DownloadBackupRecord(c *gin.Context) {
	record, ok := backupRecordFromParam(c)
	if !ok {
		return
	}
	auditFromContext(c, "offline_backup_record_download", "backup", record.BackupNo, "success", "")
	c.FileAttachment(record.FilePath, record.FileName)
}

func VerifyBackup(c *gin.Context) {
	record, ok := backupRecordFromParam(c)
	if !ok {
		return
	}
	hash, err := fileHash(record.FilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "备份校验失败: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "备份校验完成", "data": gin.H{"valid": hash == record.FileHash, "fileHash": hash}})
}

func RestoreBackup(c *gin.Context) {
	record, ok := backupRecordFromParam(c)
	if !ok {
		return
	}
	var req restoreBackupRequest
	_ = c.ShouldBindJSON(&req)
	if record.Encrypted && strings.TrimSpace(req.Password) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "冷备份恢复需要密码"})
		return
	}
	preRestore, err := createPreRestoreSnapshot(usernameFromContext(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建恢复前快照失败: " + err.Error()})
		return
	}
	restoreData, err := readBackupPayload(record, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "读取备份失败: " + err.Error()})
		return
	}
	if err := replaceDatabaseFile(restoreData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "恢复数据库失败: " + err.Error()})
		return
	}
	now := time.Now().Unix()
	record.RestoredBy = usernameFromContext(c)
	record.RestoredAt = &now
	record.Status = "restored"
	if err := storagedb.GetDB().Save(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新恢复记录失败: " + err.Error()})
		return
	}
	auditFromContext(c, "offline_backup_restore", "backup", record.BackupNo, "success", "")
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "备份恢复成功", "data": gin.H{"backup": record, "preRestoreSnapshot": preRestore}})
}

func backupRecordFromParam(c *gin.Context) (model.BackupRecord, bool) {
	var record model.BackupRecord
	if err := storagedb.GetDB().First(&record, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "备份不存在"})
		return record, false
	}
	return record, true
}

func createOfflineHotBackupRecord(c *gin.Context) (*model.BackupRecord, error) {
	if _, err := os.Stat(offlineDBPath); err != nil {
		return nil, errors.New("备份源数据库不存在")
	}
	if err := os.MkdirAll("backups", 0755); err != nil {
		return nil, err
	}
	no := newBackupNo("BACKUP")
	name := no + ".db"
	dst := filepath.Join("backups", name)
	if err := copyFile(offlineDBPath, dst); err != nil {
		return nil, err
	}
	hash, err := fileHash(dst)
	if err != nil {
		return nil, err
	}
	record := &model.BackupRecord{
		BackupNo: no, BackupType: "hot", FileName: name, FilePath: dst,
		FileHash: hash, Encrypted: false, CreatedBy: usernameFromContext(c), Status: "created",
	}
	if err := storagedb.GetDB().Create(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func createOfflineColdBackupRecord(c *gin.Context, password string) (*model.BackupRecord, error) {
	if strings.TrimSpace(password) == "" {
		return nil, errors.New("冷备份密码不能为空")
	}
	plain, err := os.ReadFile(offlineDBPath)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll("backups", 0755); err != nil {
		return nil, err
	}
	no := newBackupNo("BACKUP")
	name := no + ".cold.enc"
	dst := filepath.Join("backups", name)
	if err := writeEncryptedBackup(dst, plain, password); err != nil {
		return nil, err
	}
	hash, err := fileHash(dst)
	if err != nil {
		return nil, err
	}
	record := &model.BackupRecord{
		BackupNo: no, BackupType: "cold", FileName: name, FilePath: dst,
		FileHash: hash, Encrypted: true, CreatedBy: usernameFromContext(c), Status: "created",
	}
	if err := storagedb.GetDB().Create(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func createPreRestoreSnapshot(username string) (*model.BackupRecord, error) {
	if err := os.MkdirAll("backups", 0755); err != nil {
		return nil, err
	}
	no := newBackupNo("PRERESTORE")
	name := no + ".db"
	dst := filepath.Join("backups", name)
	if err := copyFile(offlineDBPath, dst); err != nil {
		return nil, err
	}
	hash, err := fileHash(dst)
	if err != nil {
		return nil, err
	}
	record := &model.BackupRecord{
		BackupNo: no, BackupType: "hot", FileName: name, FilePath: dst,
		FileHash: hash, Encrypted: false, CreatedBy: username, Status: "created",
	}
	if err := storagedb.GetDB().Create(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func readBackupPayload(record model.BackupRecord, password string) ([]byte, error) {
	if record.Encrypted {
		return readEncryptedBackup(record.FilePath, password)
	}
	return os.ReadFile(record.FilePath)
}

func replaceDatabaseFile(data []byte) error {
	tmpPath := offlineDBPath + ".restore.tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return err
	}
	storagedb.Close()
	if err := os.Rename(tmpPath, offlineDBPath); err != nil {
		_ = os.Remove(tmpPath)
		_ = storagedb.Init()
		return err
	}
	return storagedb.Init()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func writeEncryptedBackup(path string, plain []byte, password string) error {
	salt := make([]byte, 16)
	nonce := make([]byte, 12)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	key := deriveBackupKey(password, salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	plainHash := sha256.Sum256(plain)
	envelope := encryptedBackupFile{
		Version:    1,
		KDF:        "sha256-100000",
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(gcm.Seal(nil, nonce, plain, nil)),
		PlainHash:  hex.EncodeToString(plainHash[:]),
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func readEncryptedBackup(path string, password string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var envelope encryptedBackupFile
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	salt, err := base64.StdEncoding.DecodeString(envelope.Salt)
	if err != nil {
		return nil, err
	}
	nonce, err := base64.StdEncoding.DecodeString(envelope.Nonce)
	if err != nil {
		return nil, err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(envelope.Ciphertext)
	if err != nil {
		return nil, err
	}
	key := deriveBackupKey(password, salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("备份密码错误或文件已损坏")
	}
	sum := sha256.Sum256(plain)
	if hex.EncodeToString(sum[:]) != envelope.PlainHash {
		return nil, errors.New("备份明文哈希校验失败")
	}
	return plain, nil
}

func deriveBackupKey(password string, salt []byte) []byte {
	data := append([]byte(password), salt...)
	sum := sha256.Sum256(data)
	for i := 0; i < 100000; i++ {
		next := sha256.Sum256(sum[:])
		sum = next
	}
	return sum[:]
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func newBackupNo(prefix string) string {
	return fmt.Sprintf("%s-%s-%06d", prefix, time.Now().UTC().Format("20060102-150405"), time.Now().UTC().Nanosecond()/1000)
}

func taskDTO(task *model.OfflineTask) offlineTaskResponse {
	return offlineTaskResponse{
		ID:             task.ID,
		TaskNo:         task.TaskNo,
		TaskType:       task.TaskType,
		SourceSystem:   task.SourceSystem,
		PayloadHash:    task.PayloadHash,
		ResultHash:     task.ResultHash,
		RawPackagePath: task.RawPackagePath,
		Status:         task.Status,
		CreatedAt:      task.CreatedAt,
		UpdatedAt:      task.UpdatedAt,
	}
}

func offlineKeyDTO(key *model.OfflineKey, shards []model.KeyShard) offlineKeyResponse {
	return offlineKeyResponse{
		ID:              key.ID,
		OfflineKeyID:    key.OfflineKeyID,
		TaskNo:          key.TaskNo,
		CaseNo:          key.CaseNo,
		Address:         key.Address,
		CoinType:        key.CoinType,
		Algorithm:       key.Algorithm,
		RequiredSigners: key.RequiredSigners,
		TotalParties:    key.TotalParties,
		PublicKey:       key.PublicKey,
		LogicalOwner:    key.LogicalOwner,
		Status:          key.Status,
		CreatedAt:       key.CreatedAt,
		UpdatedAt:       key.UpdatedAt,
		Shards:          shardDTOs(shards),
	}
}

func shardDTOs(shards []model.KeyShard) []keyShardResponse {
	responses := make([]keyShardResponse, 0, len(shards))
	for _, shard := range shards {
		sum := sha256.Sum256([]byte(shard.EncryptedBlob))
		responses = append(responses, keyShardResponse{
			ShardID:             shard.ShardID,
			OfflineKeyID:        shard.OfflineKeyID,
			Address:             shard.Address,
			Username:            shard.Username,
			ShardIndex:          shard.ShardIndex,
			RecordID:            shard.RecordID,
			SeCPLC:              shard.SeCPLC,
			BlobType:            shard.BlobType,
			KeyVersion:          shard.KeyVersion,
			Status:              shard.Status,
			EncryptedBlobSize:   len(shard.EncryptedBlob),
			EncryptedBlobSHA256: "sha256:" + hex.EncodeToString(sum[:]),
			CreatedAt:           shard.CreatedAt,
			UpdatedAt:           shard.UpdatedAt,
		})
	}
	return responses
}

func shardDTOsWithKeyInfo(shards []model.KeyShard) []keyShardResponse {
	responses := shardDTOs(shards)
	for i := range responses {
		if key, err := offlineKeyStorage.GetOfflineKeyByID(responses[i].OfflineKeyID); err == nil {
			responses[i].CaseNo = key.CaseNo
			responses[i].TaskNo = key.TaskNo
			responses[i].RequiredSigners = key.RequiredSigners
			responses[i].TotalParties = key.TotalParties
		}
	}
	return responses
}

func shardDTOWithKeyInfo(shard model.KeyShard) keyShardResponse {
	return shardDTOsWithKeyInfo([]model.KeyShard{shard})[0]
}

func filterShards(shards []model.KeyShard, c *gin.Context) []model.KeyShard {
	address := strings.TrimSpace(c.Query("address"))
	username := strings.TrimSpace(c.Query("username"))
	status := strings.TrimSpace(c.Query("status"))
	if address == "" && username == "" && status == "" {
		return shards
	}
	filtered := make([]model.KeyShard, 0, len(shards))
	for _, shard := range shards {
		if address != "" && !strings.EqualFold(shard.Address, address) {
			continue
		}
		if username != "" && shard.Username != username {
			continue
		}
		if status != "" && string(shard.Status) != status {
			continue
		}
		filtered = append(filtered, shard)
	}
	return filtered
}

func auditLogDTOs(logs []model.AuditLog) []auditLogResponse {
	responses := make([]auditLogResponse, 0, len(logs))
	for _, log := range logs {
		responses = append(responses, auditLogResponse{
			ID:                log.ID,
			CreatedAt:         log.CreatedAt,
			Username:          log.Username,
			Role:              log.Role,
			Action:            log.Action,
			ResourceType:      log.ResourceType,
			ResourceID:        log.ResourceID,
			Result:            log.Result,
			ErrorMessage:      log.ErrorMessage,
			RedactedDetail:    log.RedactedDetail,
			SensitiveRedacted: log.SensitiveRedacted,
		})
	}
	return responses
}

func approvalDTOs(approvals []model.Approval) []approvalResponse {
	responses := make([]approvalResponse, 0, len(approvals))
	for _, approval := range approvals {
		responses = append(responses, approvalResponse{
			ID:          approval.ID,
			CreatedAt:   approval.CreatedAt,
			ApprovalID:  approval.ApprovalID,
			Operation:   approval.Operation,
			ResourceID:  approval.ResourceID,
			RequestedBy: approval.RequestedBy,
			ApprovedBy:  approval.ApprovedBy,
			Role:        approval.Role,
			Status:      approval.Status,
		})
	}
	return responses
}

func participationType(action string) string {
	switch {
	case strings.Contains(action, "keygen"):
		return "keygen"
	case strings.Contains(action, "sign"):
		return "sign"
	case strings.Contains(action, "destroy"):
		return "destroy"
	case strings.Contains(action, "transfer"):
		return "transfer"
	default:
		return "operation"
	}
}

func readPackageBody(c *gin.Context) ([]byte, error) {
	file, _, err := c.Request.FormFile("file")
	if err == nil {
		defer file.Close()
		return io.ReadAll(file)
	}
	if c.Request.Body == nil {
		return nil, errors.New("未提供任务包")
	}
	return io.ReadAll(c.Request.Body)
}

func validateOfflineTaskPackage(pkg offlinePackage) error {
	if pkg.SchemaVersion != "1.0" {
		return errors.New("不支持的schema_version")
	}
	if pkg.PackageType != "offline_task" {
		return errors.New("package_type必须是offline_task")
	}
	if pkg.TaskType != "custody_keygen" && pkg.TaskType != "sign" {
		return errors.New("不支持的task_type")
	}
	if pkg.TaskNo == "" {
		return errors.New("task_no不能为空")
	}
	if pkg.SourceSystem != "online" || pkg.TargetSystem != "offline" {
		return errors.New("任务包方向必须是online到offline")
	}
	if len(pkg.Payload) == 0 || pkg.PayloadHash == "" {
		return errors.New("payload和payload_hash不能为空")
	}
	return nil
}

func saveRawPackage(taskNo string, raw []byte) (string, error) {
	dir := filepath.Join("data", "offline_tasks")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("创建任务包目录失败: %w", err)
	}
	path := filepath.Join(dir, "offline_task_"+sanitizeFilePart(taskNo)+".json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		return "", fmt.Errorf("保存任务包失败: %w", err)
	}
	return path, nil
}

func loadTaskPackage(taskNo string) (*model.OfflineTask, offlinePackage, map[string]any, error) {
	task, err := offlineTaskStorage.GetTask(taskNo)
	if err != nil {
		return nil, offlinePackage{}, nil, errors.New("任务不存在")
	}
	raw, err := os.ReadFile(task.RawPackagePath)
	if err != nil {
		return nil, offlinePackage{}, nil, errors.New("任务包原文不存在")
	}
	var pkg offlinePackage
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return nil, offlinePackage{}, nil, errors.New("任务包原文格式错误")
	}
	var payload map[string]any
	if err := json.Unmarshal(pkg.Payload, &payload); err != nil {
		return nil, offlinePackage{}, nil, errors.New("任务payload格式错误")
	}
	return task, pkg, payload, nil
}

func hashPayload(raw json.RawMessage) (string, error) {
	var payload any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return "", fmt.Errorf("payload不是合法JSON: %w", err)
	}
	canonical, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("payload规范化失败: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func hashPayloadCompact(raw json.RawMessage) (string, error) {
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		return "", fmt.Errorf("payload不是合法JSON: %w", err)
	}
	sum := sha256.Sum256(buf.Bytes())
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func payloadMap(raw json.RawMessage) map[string]any {
	var payload map[string]any
	_ = json.Unmarshal(raw, &payload)
	if payload == nil {
		return map[string]any{}
	}
	return payload
}

func buildResultPackage(taskPkg offlinePackage, taskType string, payload map[string]any, createdBy string) (gin.H, string, error) {
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}
	payloadHash, err := hashPayload(rawPayload)
	if err != nil {
		return nil, "", err
	}
	return gin.H{
		"schema_version": "1.0",
		"package_type":   "offline_result",
		"task_type":      taskType,
		"task_no":        taskPkg.TaskNo,
		"source_system":  "offline",
		"target_system":  "online",
		"created_by":     createdBy,
		"created_at":     time.Now().UTC().Format(time.RFC3339),
		"payload":        payload,
		"payload_hash":   payloadHash,
		"package_signature": gin.H{
			"algorithm": "",
			"key_id":    "",
			"signature": "",
		},
	}, payloadHash, nil
}

func auditFromContext(c *gin.Context, action, resourceType, resourceID, result, detail string) {
	_ = offlineAuditStore.CreateAuditLog(model.AuditLog{
		Username:       usernameFromContext(c),
		Role:           roleFromContext(c),
		Action:         action,
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		Result:         result,
		RedactedDetail: detail,
	})
}

func usernameFromContext(c *gin.Context) string {
	if value, ok := c.Get("userName"); ok {
		if username, ok := value.(string); ok {
			return username
		}
	}
	return ""
}

func roleFromContext(c *gin.Context) string {
	if value, ok := c.Get("role"); ok {
		if role, ok := value.(string); ok {
			return role
		}
	}
	return ""
}

func auditFilterFromQuery(c *gin.Context) storage.AuditLogFilter {
	page := intFromQuery(c, "page", 1)
	pageSize := firstIntFromQuery(c, []string{"pageSize", "page_size", "limit"}, 20)
	return storage.AuditLogFilter{
		Limit:    pageSize,
		Page:     page,
		PageSize: pageSize,
		TimeFrom: timeFromQuery(c, "time_from"),
		TimeTo:   timeFromQuery(c, "time_to"),
		Username: strings.TrimSpace(c.Query("username")),
		Role:     strings.TrimSpace(c.Query("role")),
		Action:   strings.TrimSpace(c.Query("action")),
		Resource: strings.TrimSpace(c.Query("resource")),
		CaseNo:   strings.TrimSpace(c.Query("case_no")),
		Address:  strings.TrimSpace(c.Query("address")),
		Result:   strings.TrimSpace(c.Query("result")),
	}
}

func intFromPayload(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	default:
		return 0
	}
}

func intFromQuery(c *gin.Context, key string, fallback int) int {
	value := c.Query(key)
	if value == "" {
		return fallback
	}
	var parsed int
	if _, err := fmt.Sscanf(value, "%d", &parsed); err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func firstIntFromQuery(c *gin.Context, keys []string, fallback int) int {
	for _, key := range keys {
		if c.Query(key) != "" {
			return intFromQuery(c, key, fallback)
		}
	}
	return fallback
}

func timeFromQuery(c *gin.Context, key string) time.Time {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed
	}
	if parsed, err := time.Parse("2006-01-02 15:04:05", value); err == nil {
		return parsed
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return parsed
	}
	return time.Time{}
}

func stringFromPayload(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprint(value)
}

func sanitizeFilePart(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "task"
	}
	return b.String()
}
