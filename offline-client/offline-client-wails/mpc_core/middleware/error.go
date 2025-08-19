package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorHandler 错误处理中间件
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// 如果已经有响应，则不再处理
		if c.Writer.Status() != http.StatusOK {
			return
		}

		// 检查是否有错误
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err
			// 根据错误类型返回不同的HTTP状态码
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	}
}
