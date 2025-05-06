package handler

import (
	"net/http"
	"offline-server/storage"

	"github.com/gin-gonic/gin"
)

// CreateSeRequest 创建SE的请求结构
type CreateSeRequest struct {
	SeId string `json:"seId" binding:"required"`
	CPIC string `json:"cpic" binding:"required"`
}

// CreateSe 创建新的安全芯片记录
func CreateSe(c *gin.Context) {
	var req CreateSeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":  400,
			"error": "无效的请求参数",
		})
		return
	}

	seStorage := storage.GetSeStorage()
	se, err := seStorage.CreateSe(req.SeId, req.CPIC)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":  500,
			"error": "创建安全芯片记录失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": se,
	})
}
