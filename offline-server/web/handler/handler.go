package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 处理请求参数绑定的通用函数
func bindJSON(c *gin.Context, req interface{}) bool {
	if err := c.BindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return false
	}
	return true
}
