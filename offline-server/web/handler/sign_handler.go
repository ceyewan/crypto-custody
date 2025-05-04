// Package handler 提供 Web API 的请求处理函数
package handler

import (
	"net/http"
	"offline-server/web/service"

	"github.com/gin-gonic/gin"
)

// 创建签名会话 key
func CreateSignSessionKey(c *gin.Context) {
	// 从URL参数获取initiator
	initiator := c.Param("initiator")
	if initiator == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少发起者参数"})
		return
	}

	// 调用服务层创建签名会话
	sessionKey, err := service.CreateSignSessionKey(initiator)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"session_key": sessionKey})
}
