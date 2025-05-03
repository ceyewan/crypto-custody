// Package handler 提供 Web API 的请求处理函数
package handler

import (
	"net/http"
	"offline-server/web/service"

	"github.com/gin-gonic/gin"
)

// 创建签名会话 key
func CreateSignKey(c *gin.Context) {
	// 获取请求参数
	var req struct {
		Initiator string `json:"initiator" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参数错误"})
		return
	}

	// 调用服务层创建密钥生成会话
	sessionKey, err := service.CreateSignSessionKey(req.Initiator)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"session_key": sessionKey})
}
