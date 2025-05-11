package handlers

import (
	"net/http"
	"strconv"

	"online-server/service"

	"github.com/gin-gonic/gin"
)

// GetTransaction 获取交易详情
func GetTransaction(c *gin.Context) {
	txIDStr := c.Param("id")
	txID, err := strconv.ParseUint(txIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的交易ID"})
		return
	}

	accountService, err := service.GetAccountServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "初始化账户服务失败: " + err.Error()})
		return
	}

	tx, err := accountService.GetTransaction(uint(txID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取交易详情失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, tx)
}

// GetTransactionByHash 通过消息哈希获取交易
func GetTransactionByHash(c *gin.Context) {
	messageHash := c.Param("hash")
	if messageHash == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "消息哈希不能为空"})
		return
	}

	accountService, err := service.GetAccountServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "初始化账户服务失败: " + err.Error()})
		return
	}

	tx, err := accountService.GetTransactionByMessageHash(messageHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取交易详情失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, tx)
}

// GetUserTransactions 获取用户的交易历史
func GetUserTransactions(c *gin.Context) {
	address := c.Param("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "地址不能为空"})
		return
	}

	accountService, err := service.GetAccountServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "初始化账户服务失败: " + err.Error()})
		return
	}

	txs, err := accountService.GetUserTransactions(address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取交易历史失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, txs)
}
