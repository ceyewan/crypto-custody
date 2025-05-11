package handlers

import (
	"math/big"
	"net/http"
	"strconv"

	"online-server/ethereum"

	"github.com/gin-gonic/gin"
)

// PrepareTransaction 准备交易数据以供签名
func PrepareTransaction(c *gin.Context) {
	var input struct {
		FromAddress string  `json:"from" binding:"required"`
		ToAddress   string  `json:"to" binding:"required"`
		Amount      float64 `json:"amount" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ethService, err := ethereum.GetInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "初始化以太坊服务失败: " + err.Error()})
		return
	}

	// 将输入的浮点数转换为big.Float
	amount := new(big.Float)
	amount.SetFloat64(input.Amount)

	// 创建交易
	tx, messageHash, err := ethService.CreateTransaction(input.FromAddress, input.ToAddress, amount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建交易失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transaction_id": tx.ID,
		"message_hash":   messageHash,
		"from":           input.FromAddress,
		"to":             input.ToAddress,
		"amount":         input.Amount,
		"status":         "待签名",
	})
}

// SignTransaction 接收签名并处理交易
func SignTransaction(c *gin.Context) {
	var input struct {
		MessageHash string `json:"message_hash" binding:"required"`
		Signature   string `json:"signature" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ethService, err := ethereum.GetInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "初始化以太坊服务失败: " + err.Error()})
		return
	}

	// 处理签名
	tx, err := ethService.SignTransaction(input.MessageHash, input.Signature)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "处理签名失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transaction_id": tx.ID,
		"message_hash":   input.MessageHash,
		"status":         "已签名",
		"tx_hash":        tx.TxHash,
	})
}

// ProcessTransaction 发送已签名的交易
func ProcessTransaction(c *gin.Context) {
	txIDStr := c.Param("id")
	txID, err := strconv.ParseUint(txIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的交易ID"})
		return
	}

	ethService, err := ethereum.GetInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "初始化以太坊服务失败: " + err.Error()})
		return
	}

	// 发送交易
	tx, err := ethService.SendTransaction(uint(txID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "发送交易失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"transaction_id": tx.ID,
		"tx_hash":        tx.TxHash,
		"status":         "已提交",
		"message":        "交易已提交到网络，请等待确认",
	})
}

// GetTransactionStatus 获取交易状态
func GetTransactionStatus(c *gin.Context) {
	txIDStr := c.Param("id")
	txID, err := strconv.ParseUint(txIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的交易ID"})
		return
	}

	ethService, err := ethereum.GetInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "初始化以太坊服务失败: " + err.Error()})
		return
	}

	// 获取交易状态
	tx, err := ethService.GetTransactionStatus(uint(txID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取交易状态失败: " + err.Error()})
		return
	}

	response := gin.H{
		"transaction_id": tx.ID,
		"from":           tx.FromAddress,
		"to":             tx.ToAddress,
		"value":          tx.Value,
		"status":         tx.Status,
		"created_at":     tx.CreatedAt,
	}

	if tx.TxHash != "" {
		response["tx_hash"] = tx.TxHash
	}

	if tx.SubmittedAt != nil {
		response["submitted_at"] = tx.SubmittedAt
	}

	if tx.ConfirmedAt != nil {
		response["confirmed_at"] = tx.ConfirmedAt
		response["block_number"] = tx.BlockNumber
		response["block_hash"] = tx.BlockHash
	}

	if tx.Error != "" {
		response["error"] = tx.Error
	}

	c.JSON(http.StatusOK, response)
}

// GetUserTransactions 获取用户的交易历史
func GetUserTransactions(c *gin.Context) {
	address := c.Param("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "地址不能为空"})
		return
	}

	ethService, err := ethereum.GetInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "初始化以太坊服务失败: " + err.Error()})
		return
	}

	// 获取用户交易历史
	txs, err := ethService.GetUserTransactions(address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取交易历史失败: " + err.Error()})
		return
	}

	// 转换为响应格式
	var response []gin.H
	for _, tx := range txs {
		txData := gin.H{
			"transaction_id": tx.ID,
			"from":           tx.FromAddress,
			"to":             tx.ToAddress,
			"value":          tx.Value,
			"status":         tx.Status,
			"created_at":     tx.CreatedAt,
		}

		if tx.TxHash != "" {
			txData["tx_hash"] = tx.TxHash
		}

		if tx.SubmittedAt != nil {
			txData["submitted_at"] = tx.SubmittedAt
		}

		if tx.ConfirmedAt != nil {
			txData["confirmed_at"] = tx.ConfirmedAt
			txData["block_number"] = tx.BlockNumber
			txData["block_hash"] = tx.BlockHash
		}

		response = append(response, txData)
	}

	c.JSON(http.StatusOK, response)
}

// CheckPendingTransactions 手动检查所有待处理的交易
func CheckPendingTransactions(c *gin.Context) {
	ethService, err := ethereum.GetInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "初始化以太坊服务失败: " + err.Error()})
		return
	}

	err = ethService.CheckPendingTransactions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "检查待处理交易失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "所有待处理交易已检查"})
}
