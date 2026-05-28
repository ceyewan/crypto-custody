package handler

import (
	"bytes"
	"crypto/sha256"
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
	if pkg.PayloadHash != payloadHash {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payload_hash不匹配"})
		return
	}

	if existing, err := offlineTaskStorage.GetTask(pkg.TaskNo); err == nil {
		if existing.PayloadHash != payloadHash {
			c.JSON(http.StatusConflict, gin.H{"error": "任务编号已存在但payload_hash不同"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "task": taskDTO(existing), "duplicated": true})
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
	c.JSON(http.StatusOK, gin.H{"success": true, "task": taskDTO(task)})
}

// GetOfflineTask 查询导入任务摘要。
func GetOfflineTask(c *gin.Context) {
	taskNo := c.Param("task_no")
	task, err := offlineTaskStorage.GetTask(taskNo)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "任务不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "task": taskDTO(task)})
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
			"offline_key_id":   req.OfflineKeyID,
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

// TransferOfflineKey 只调整离线系统业务归属，不移动 SE 内 AES key。
func TransferOfflineKey(c *gin.Context) {
	id := c.Param("offline_key_id")
	var req struct {
		NewOwner string `json:"new_owner"`
		Reason   string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || strings.TrimSpace(req.NewOwner) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "new_owner不能为空"})
		return
	}
	if err := offlineKeyStorage.UpdateOfflineKeyOwner(id, req.NewOwner); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "离线密钥不存在"})
		return
	}
	createApprovedRecord(c, "offline_key_transfer", id)
	auditFromContext(c, "offline_key_transfer", "offline_key", id, "success", req.Reason)
	c.JSON(http.StatusOK, gin.H{"success": true})
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
	if key.Status != model.OfflineKeyStatusActive {
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

	createApprovedRecord(c, "offline_key_destroy", id)
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

// ListAuditLogs 查询脱敏审计日志。
func ListAuditLogs(c *gin.Context) {
	logs, err := offlineAuditStore.ListAuditLogs(intFromQuery(c, "limit", 100))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询审计日志失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "logs": auditLogDTOs(logs)})
}

// ListApprovals 查询敏感操作审批记录。
func ListApprovals(c *gin.Context) {
	approvals, err := offlineApproval.ListApprovals(intFromQuery(c, "limit", 100))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询审批记录失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "approvals": approvalDTOs(approvals)})
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
	var buf bytes.Buffer
	if err := json.Compact(&buf, raw); err != nil {
		return "", fmt.Errorf("payload不是合法JSON: %w", err)
	}
	sum := sha256.Sum256(buf.Bytes())
	return "sha256:" + hex.EncodeToString(sum[:]), nil
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

func createApprovedRecord(c *gin.Context, operation, resourceID string) {
	username := usernameFromContext(c)
	_, _ = offlineApproval.CreateApproval(model.Approval{
		ApprovalID:  fmt.Sprintf("APPROVAL-%s-%d", sanitizeFilePart(resourceID), time.Now().UnixNano()),
		Operation:   operation,
		ResourceID:  resourceID,
		RequestedBy: username,
		ApprovedBy:  username,
		Role:        roleFromContext(c),
		Status:      model.ApprovalApproved,
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
