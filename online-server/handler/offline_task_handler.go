package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"online-server/dto"
	"online-server/model"
	"online-server/service"
	"online-server/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func CreateCustodyKeygenTask(c *gin.Context) {
	var req dto.CustodyKeygenTaskRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	var cs model.Case
	if err := utils.GetDB().First(&cs, req.CaseID).Error; err != nil {
		utils.ResponseWithError(c, http.StatusNotFound, "案件不存在")
		return
	}
	coinType := req.CoinType
	if coinType == "" {
		coinType = "ETH"
	}
	policy := req.ThresholdPolicy
	if policy == "" {
		policy = "2_of_3"
	}
	taskNo := service.NewBusinessNo("TASK")
	payload := gin.H{
		"case_no":   cs.CaseNo,
		"coin_type": coinType,
		"chain_id":  "1",
		"threshold_policy": gin.H{
			"required_signers": 2,
			"total_parties":    3,
			"policy":           policy,
		},
		"business_reason": "创建案件托管钱包",
	}
	hash := hashPayload(payload)
	createdAt := time.Now().UTC().Format(time.RFC3339)
	taskPackage := buildOfflineTaskPackage(taskNo, string(model.OfflineTaskCustodyKeygen), c.GetString("Username"), createdAt, payload, hash)
	now := time.Now().Unix()
	task := model.OfflineTask{
		TaskNo: taskNo, TaskType: model.OfflineTaskCustodyKeygen,
		CaseID: &cs.ID, CaseNo: cs.CaseNo,
		PayloadHash: hash, Status: model.OfflineTaskExported,
		ExportedBy: c.GetString("Username"), ExportedAt: &now,
	}
	if err := utils.GetDB().Create(&task).Error; err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "创建离线任务失败: "+err.Error())
		return
	}
	service.AuditAction(c, "offline_task.create_custody_keygen", "offline_task", strconv.FormatUint(uint64(task.ID), 10), cs.CaseNo, "success", "", taskPackage)
	utils.ResponseWithData(c, "托管钱包生成任务创建成功", gin.H{"task": task, "payload": payload, "package": taskPackage})
}

func ExportOfflineTask(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}
	var task model.OfflineTask
	if err := utils.GetDB().First(&task, id).Error; err != nil {
		utils.ResponseWithError(c, http.StatusNotFound, "离线任务不存在")
		return
	}
	payload, err := offlineTaskPayload(task)
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	taskPackage := buildOfflineTaskPackage(task.TaskNo, string(task.TaskType), task.ExportedBy, time.Now().UTC().Format(time.RFC3339), payload, hashPayload(payload))
	service.AuditAction(c, "offline_task.export", "offline_task", strconv.FormatUint(uint64(task.ID), 10), task.CaseNo, "success", "", taskPackage)
	utils.ResponseWithData(c, "离线任务导出成功", taskPackage)
}

func ImportOfflineTaskResult(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}
	var req dto.ImportOfflineResultRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	var task model.OfflineTask
	if err := utils.GetDB().First(&task, id).Error; err != nil {
		utils.ResponseWithError(c, http.StatusNotFound, "离线任务不存在")
		return
	}
	now := time.Now().Unix()
	task.ResultHash = hashObject(req.Result)
	task.Status = model.OfflineTaskImported
	task.ImportedBy = c.GetString("Username")
	task.ImportedAt = &now
	if err := utils.GetDB().Save(&task).Error; err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "导入离线结果失败: "+err.Error())
		return
	}
	service.AuditAction(c, "offline_task.import_result", "offline_task", strconv.FormatUint(uint64(task.ID), 10), task.CaseNo, "success", "", req.Result)
	utils.ResponseWithData(c, "离线结果导入成功", task)
}

func ListOfflineTasks(c *gin.Context) {
	var req dto.OfflineTaskListRequest
	_ = c.ShouldBindQuery(&req)
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}
	query := utils.GetDB().Model(&model.OfflineTask{})
	if req.TaskType != "" {
		query = query.Where("task_type = ?", req.TaskType)
	}
	if req.CaseNo != "" {
		query = query.Where("case_no = ?", req.CaseNo)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	var total int64
	_ = query.Count(&total).Error
	var tasks []model.OfflineTask
	if err := query.Order("created_at DESC").Limit(req.PageSize).Offset((req.Page - 1) * req.PageSize).Find(&tasks).Error; err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "查询离线任务失败: "+err.Error())
		return
	}
	utils.ResponseWithData(c, "查询离线任务成功", gin.H{"items": tasks, "total": total, "page": req.Page, "pageSize": req.PageSize})
}

func GetOfflineTask(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}
	var task model.OfflineTask
	if err := utils.GetDB().First(&task, id).Error; err != nil {
		utils.ResponseWithError(c, http.StatusNotFound, "离线任务不存在")
		return
	}
	utils.ResponseWithData(c, "查询离线任务成功", task)
}

func hashObject(v interface{}) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func hashPayload(v interface{}) string {
	b, _ := json.Marshal(v)
	var decoded interface{}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.UseNumber()
	_ = decoder.Decode(&decoded)
	canonical, _ := json.Marshal(decoded)
	sum := sha256.Sum256(canonical)
	return fmt.Sprintf("sha256:%s", hex.EncodeToString(sum[:]))
}

func buildOfflineTaskPackage(taskNo, taskType, createdBy, createdAt string, payload gin.H, payloadHash string) gin.H {
	return gin.H{
		"schema_version": "1.0",
		"package_type":   "offline_task",
		"task_type":      taskType,
		"task_no":        taskNo,
		"source_system":  "online",
		"target_system":  "offline",
		"created_by":     createdBy,
		"created_at":     createdAt,
		"payload":        payload,
		"payload_hash":   payloadHash,
		"package_signature": gin.H{
			"algorithm": "",
			"key_id":    "",
			"signature": "",
		},
	}
}

func offlineTaskPayload(task model.OfflineTask) (gin.H, error) {
	if task.TaskType == model.OfflineTaskSign && task.TransactionID != nil {
		var tx model.Transaction
		if err := utils.GetDB().First(&tx, *task.TransactionID).Error; err != nil {
			return nil, fmt.Errorf("查询交易失败: %w", err)
		}
		return signTaskPayload(tx), nil
	}
	return gin.H{
		"case_no":   task.CaseNo,
		"coin_type": "ETH",
		"chain_id":  "1",
		"threshold_policy": gin.H{
			"required_signers": 2,
			"total_parties":    3,
			"policy":           "2_of_3",
		},
		"business_reason": "创建案件托管钱包",
	}, nil
}

func signTaskPayload(tx model.Transaction) gin.H {
	return gin.H{
		"case_no":        tx.CaseNo,
		"transaction_no": tx.TxNo,
		"from_address":   tx.FromAddress,
		"to_address":     tx.ToAddress,
		"value":          tx.Value,
		"coin_type":      tx.CoinType,
		"chain_id":       "1",
		"message_hash":   tx.MessageHash,
		"reason":         tx.Reason,
		"display": gin.H{
			"amount":          tx.Value,
			"recipient_label": tx.ToAddress,
		},
	}
}
