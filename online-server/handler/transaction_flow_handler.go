package handler

import (
	"encoding/hex"
	"math/big"
	"net/http"
	"online-server/dto"
	"online-server/model"
	"online-server/service"
	"online-server/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func CreateTransactionPrepared(c *gin.Context) {
	var req dto.CreateTransactionRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	txType := req.TxType
	if txType == "" {
		txType = "withdraw"
	}
	coinType := req.CoinType
	if coinType == "" {
		coinType = "ETH"
	}
	var caseID *uint
	if req.CaseID != 0 {
		caseID = &req.CaseID
	}
	var fromAccountID *uint
	if req.FromAccountID != 0 {
		fromAccountID = &req.FromAccountID
	}
	txNo, err := service.NewTransactionNo()
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "生成交易编号失败: "+err.Error())
		return
	}
	messageHash, generatedID, err := buildTransactionHash(req.FromAddress, req.ToAddress, req.Value)
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	tx := model.Transaction{
		TxNo: txNo, CaseID: caseID, CaseNo: req.CaseNo,
		TxType: txType, FromAccountID: fromAccountID, FromAddress: req.FromAddress,
		ToAddress: req.ToAddress, Value: req.Value, CoinType: coinType,
		Reason: req.Reason, MessageHash: messageHash, Status: model.StatusPending, CreatedBy: c.GetString("Username"),
	}
	if err := utils.GetDB().Create(&tx).Error; err != nil {
		utils.ResponseWithError(c, http.StatusBadRequest, "创建交易失败: "+err.Error())
		return
	}
	if err := rebindPreparedTransaction(messageHash, generatedID, tx.ID); err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	service.AuditAction(c, "transaction.create", "transaction", strconv.FormatUint(uint64(tx.ID), 10), tx.CaseNo, "success", "", tx)
	utils.ResponseWithData(c, "交易创建成功，已生成待签名哈希", tx)
}

func BatchImportTransactions(c *gin.Context) {
	var req dto.BatchImportTransactionsRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	if len(req.Transactions) == 0 {
		utils.ResponseWithError(c, http.StatusBadRequest, "没有可导入的交易")
		return
	}

	success := 0
	for _, item := range req.Transactions {
		txNo := strings.TrimSpace(item.TxNo)
		if txNo == "" {
			generated, err := service.NewTransactionNo()
			if err != nil {
				service.AuditAction(c, "transaction.batch_import", "transaction", "", item.CaseNo, "failure", err.Error(), gin.H{"total": len(req.Transactions)})
				utils.ResponseWithError(c, http.StatusInternalServerError, "生成交易编号失败: "+err.Error())
				return
			}
			txNo = generated
		}
		txType := defaultString(item.TxType, "test")
		coinType := defaultString(item.CoinType, "ETH")
		status := model.StatusDraft
		if item.Status != "" {
			if parsed, ok := parseTransactionStatus(item.Status); ok {
				status = parsed
			}
		}

		var caseID *uint
		if item.CaseID != 0 {
			caseID = &item.CaseID
		} else if item.CaseNo != "" {
			var cs model.Case
			if err := utils.GetDB().Where("case_no = ?", item.CaseNo).First(&cs).Error; err == nil {
				caseID = &cs.ID
			}
		}
		var fromAccountID *uint
		if item.FromAccountID != 0 {
			fromAccountID = &item.FromAccountID
		} else if item.FromAddress != "" {
			var account model.Account
			if err := utils.GetDB().Where("address = ?", item.FromAddress).First(&account).Error; err == nil {
				fromAccountID = &account.ID
			}
		}

		tx := model.Transaction{
			TxNo: txNo, CaseID: caseID, CaseNo: item.CaseNo,
			TxType: txType, FromAccountID: fromAccountID, FromAddress: item.FromAddress,
			ToAddress: item.ToAddress, Value: item.Value, CoinType: coinType,
			Reason: item.Reason, MessageHash: item.MessageHash, TxHash: item.TxHash,
			Status: status, CreatedBy: c.GetString("Username"),
		}
		if err := utils.GetDB().Create(&tx).Error; err != nil {
			service.AuditAction(c, "transaction.batch_import", "transaction", "", tx.CaseNo, "failure", err.Error(), gin.H{"total": len(req.Transactions)})
			utils.ResponseWithError(c, http.StatusBadRequest, "批量导入交易失败: "+err.Error())
			return
		}
		success++
	}

	service.AuditAction(c, "transaction.batch_import", "transaction", "", "", "success", "", gin.H{"total": len(req.Transactions), "success": success})
	utils.ResponseWithData(c, "批量导入交易成功", gin.H{"total": len(req.Transactions), "success": success, "failed": len(req.Transactions) - success})
}

func ListTransactionsV2(c *gin.Context) {
	var req dto.TransactionListRequest
	_ = c.ShouldBindQuery(&req)
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}
	query := utils.GetDB().Model(&model.Transaction{})
	if req.Status != "" {
		if status, ok := parseTransactionStatus(req.Status); ok {
			query = query.Where("status = ?", status)
		}
	}
	if req.CaseNo != "" {
		query = query.Where("case_no = ?", req.CaseNo)
	}
	if req.Address != "" {
		query = query.Where("from_address = ? OR to_address = ?", req.Address, req.Address)
	}
	var total int64
	_ = query.Count(&total).Error
	var txs []model.Transaction
	if err := query.Order("created_at DESC").Limit(req.PageSize).Offset((req.Page - 1) * req.PageSize).Find(&txs).Error; err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "查询交易失败: "+err.Error())
		return
	}
	utils.ResponseWithData(c, "查询交易成功", gin.H{"items": txs, "total": total, "page": req.Page, "pageSize": req.PageSize})
}

func GetTransactionV2(c *gin.Context) {
	GetTransactionByID(c)
}

func PrepareTransactionV2(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}
	var tx model.Transaction
	if err := utils.GetDB().First(&tx, id).Error; err != nil {
		utils.ResponseWithError(c, http.StatusNotFound, "交易不存在")
		return
	}
	if _, err := ensureTransactionPrepared(c, &tx); err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	utils.ResponseWithData(c, "交易构建成功", tx)
}

func ExportSignTask(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}
	var tx model.Transaction
	if err := utils.GetDB().First(&tx, id).Error; err != nil {
		utils.ResponseWithError(c, http.StatusNotFound, "交易不存在")
		return
	}
	if _, err := ensureTransactionPrepared(c, &tx); err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	taskNo, err := service.NewOfflineTaskNo()
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "生成任务编号失败: "+err.Error())
		return
	}
	payload := signTaskPayload(tx)
	payloadHash := hashPayload(payload)
	taskPackage := buildOfflineTaskPackage(taskNo, "sign", c.GetString("Username"), time.Now().UTC().Format(time.RFC3339), payload, payloadHash)
	now := time.Now().Unix()
	tx.Status = model.StatusSignatureExported
	tx.ExportedAt = &now
	_ = utils.GetDB().Save(&tx).Error
	task := model.OfflineTask{
		TaskNo: taskNo, TaskType: model.OfflineTaskSign, CaseNo: tx.CaseNo,
		TransactionID: &tx.ID, PayloadHash: payloadHash, Status: model.OfflineTaskExported,
		ExportedBy: c.GetString("Username"), ExportedAt: &now,
	}
	_ = utils.GetDB().Create(&task).Error
	service.AuditAction(c, "transaction.export_sign_task", "transaction", strconv.FormatUint(uint64(tx.ID), 10), tx.CaseNo, "success", "", taskPackage)
	utils.ResponseWithData(c, "签名任务导出成功", gin.H{"task": task, "payload": payload, "package": taskPackage})
}

func ensureTransactionPrepared(c *gin.Context, tx *model.Transaction) (bool, error) {
	if tx.MessageHash != "" {
		return false, nil
	}
	messageHash, generatedID, err := buildTransactionHash(tx.FromAddress, tx.ToAddress, tx.Value)
	if err != nil {
		return false, err
	}
	tx.MessageHash = messageHash
	tx.Status = model.StatusPending
	if err := utils.GetDB().Save(tx).Error; err != nil {
		return false, errWithPrefix("保存交易哈希失败", err)
	}
	if err := rebindPreparedTransaction(messageHash, generatedID, tx.ID); err != nil {
		return false, err
	}
	service.AuditAction(c, "transaction.prepare", "transaction", strconv.FormatUint(uint64(tx.ID), 10), tx.CaseNo, "success", "", tx)
	return true, nil
}

func buildTransactionHash(fromAddress, toAddress, value string) (string, uint, error) {
	amountText := strings.Fields(value)
	amount := new(big.Float)
	if len(amountText) > 0 {
		if _, ok := amount.SetString(amountText[0]); !ok {
			return "", 0, errWithMessage("交易金额格式错误: " + value)
		}
	} else {
		if _, ok := amount.SetString(value); !ok {
			return "", 0, errWithMessage("交易金额格式错误: " + value)
		}
	}
	tm, err := getTransactionManager()
	if err != nil {
		return "", 0, errWithPrefix("获取交易管理器失败", err)
	}
	generatedID, messageHash, err := tm.CreateTransaction(fromAddress, toAddress, amount)
	if err != nil {
		return "", 0, errWithPrefix("构建交易失败", err)
	}
	return messageHash, generatedID, nil
}

func rebindPreparedTransaction(messageHash string, generatedID uint, transactionID uint) error {
	tm, err := getTransactionManager()
	if err != nil {
		return errWithPrefix("获取交易管理器失败", err)
	}
	if err := tm.RebindTransactionRecord(messageHash, transactionID); err != nil {
		return errWithPrefix("绑定待签名交易失败", err)
	}
	if generatedID != 0 && generatedID != transactionID {
		if err := utils.GetDB().Unscoped().Delete(&model.Transaction{}, generatedID).Error; err != nil {
			return errWithPrefix("清理临时交易记录失败", err)
		}
	}
	return nil
}

func errWithPrefix(prefix string, err error) error {
	return &prefixedError{prefix: prefix, err: err}
}

func errWithMessage(message string) error {
	return &prefixedError{prefix: message}
}

type prefixedError struct {
	prefix string
	err    error
}

func (e *prefixedError) Error() string {
	if e.err == nil {
		return e.prefix
	}
	return e.prefix + ": " + e.err.Error()
}

func ImportTransactionSignature(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}
	var req dto.ImportSignatureRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	var tx model.Transaction
	if err := utils.GetDB().First(&tx, id).Error; err != nil {
		utils.ResponseWithError(c, http.StatusNotFound, "交易不存在")
		return
	}
	sig := strings.TrimPrefix(req.Signature, "0x")
	sigBytes, err := hex.DecodeString(sig)
	if err != nil {
		utils.ResponseWithError(c, http.StatusBadRequest, "签名格式错误: "+err.Error())
		return
	}
	if tx.MessageHash != "" {
		tm, err := getTransactionManager()
		if err != nil {
			utils.ResponseWithError(c, http.StatusInternalServerError, "获取交易管理器失败: "+err.Error())
			return
		}
		if _, err := tm.SignTransaction(tx.MessageHash, sig); err != nil {
			utils.ResponseWithError(c, http.StatusBadRequest, "签名验证或附加失败: "+err.Error())
			return
		}
	}
	now := time.Now().Unix()
	tx.Signature = sigBytes
	tx.Status = model.StatusSigned
	tx.SignedAt = &now
	if req.MessageHash != "" {
		tx.MessageHash = req.MessageHash
	}
	if err := utils.GetDB().Save(&tx).Error; err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "保存签名失败: "+err.Error())
		return
	}
	service.AuditAction(c, "transaction.import_signature", "transaction", strconv.FormatUint(uint64(tx.ID), 10), tx.CaseNo, "success", "", gin.H{"messageHash": tx.MessageHash})
	utils.ResponseWithData(c, "签名结果导入成功", tx)
}

func BroadcastTransactionV2(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}
	var tx model.Transaction
	if err := utils.GetDB().First(&tx, id).Error; err != nil {
		utils.ResponseWithError(c, http.StatusNotFound, "交易不存在")
		return
	}
	if tx.MessageHash == "" || len(tx.Signature) == 0 {
		utils.ResponseWithError(c, http.StatusBadRequest, "交易缺少哈希或签名")
		return
	}
	// 真实广播仍由旧 TransactionManager 负责；若缓存丢失，返回明确错误，便于测试识别边界。
	tm, err := getTransactionManager()
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "获取交易管理器失败: "+err.Error())
		return
	}
	_, txHash, err := tm.SendTransaction(tx.MessageHash)
	if err != nil {
		utils.ResponseWithError(c, http.StatusBadRequest, "广播失败，请确认交易在当前服务进程中完成构建和签名: "+err.Error())
		return
	}
	now := time.Now().Unix()
	tx.TxHash = txHash
	tx.Status = model.StatusBroadcasted
	tx.BroadcastedAt = &now
	_ = utils.GetDB().Save(&tx).Error
	service.AuditAction(c, "transaction.broadcast", "transaction", strconv.FormatUint(uint64(tx.ID), 10), tx.CaseNo, "success", "", gin.H{"txHash": txHash})
	utils.ResponseWithData(c, "交易广播成功", tx)
}

func CheckTransactionReceiptV2(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}
	var tx model.Transaction
	if err := utils.GetDB().First(&tx, id).Error; err != nil {
		utils.ResponseWithError(c, http.StatusNotFound, "交易不存在")
		return
	}
	utils.ResponseWithData(c, "交易回执查询完成", tx)
}

func parseTransactionStatus(status string) (model.TransactionStatus, bool) {
	switch status {
	case "draft":
		return model.StatusDraft, true
	case "pending_signature", "prepared":
		return model.StatusPending, true
	case "signature_exported":
		return model.StatusSignatureExported, true
	case "signed":
		return model.StatusSigned, true
	case "broadcasted", "sent":
		return model.StatusBroadcasted, true
	case "confirmed":
		return model.StatusConfirmed, true
	case "failed":
		return model.StatusFailed, true
	case "cancelled":
		return model.StatusCancelled, true
	default:
		return 0, false
	}
}
