package middleware

import (
	"time"

	"web-se/utils"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LoggerMiddleware 返回一个Gin中间件，记录API请求日志
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		startTime := time.Now()

		// 处理请求
		c.Next()

		// 结束时间
		endTime := time.Now()
		// 执行时间
		latencyTime := endTime.Sub(startTime)

		// 请求方法
		reqMethod := c.Request.Method
		// 请求路由
		reqUri := c.Request.RequestURI
		// 状态码
		statusCode := c.Writer.Status()
		// 请求IP
		clientIP := c.ClientIP()

		// 构建日志字段
		fields := []zap.Field{
			utils.String("method", reqMethod),
			utils.String("uri", reqUri),
			utils.String("ip", clientIP),
			utils.Int("status", statusCode),
			utils.String("latency", latencyTime.String()),
		}

		// 根据状态码记录不同级别的日志
		switch {
		case statusCode >= 500:
			utils.LogError("API请求", fields...)
		case statusCode >= 400:
			utils.LogWarn("API请求", fields...)
		default:
			utils.LogInfo("API请求", fields...)
		}
	}
}
