package handler

import (
	"net/http"

	"online-server/dto"
	"online-server/service"

	"github.com/gin-gonic/gin"
)

// GetBalance 获取地址余额
func GetBalance(c *gin.Context) {
	address := c.Param("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "地址不能为空"})
		return
	}

	ethService, err := service.GetEthServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "初始化以太坊服务失败: " + err.Error()})
		return
	}

	balance, err := ethService.GetBalance(address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取余额失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, balance)
}

// PrepareTransaction 准备交易数据以供签名
func PrepareTransaction(c *gin.Context) {
	var req dto.TransactionRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ethService, err := service.GetEthServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "初始化以太坊服务失败: " + err.Error()})
		return
	}

	// 准备交易数据
	messageHash, err := ethService.PrepareTransaction(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "准备交易失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message_hash": messageHash,
		"from":         req.FromAddress,
		"to":           req.ToAddress,
		"amount":       req.Amount,
		"status":       "待签名",
	})
}

// SignAndSendTransaction 签名并发送交易
func SignAndSendTransaction(c *gin.Context) {
	var req dto.SignatureRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ethService, err := service.GetEthServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "初始化以太坊服务失败: " + err.Error()})
		return
	}

	// 签名并发送交易
	txHash, err := ethService.SignAndSendTransaction(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "签名并发送交易失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tx_hash": txHash,
		"status":  "已提交",
		"message": "交易已提交到网络，请等待确认",
	})
}
