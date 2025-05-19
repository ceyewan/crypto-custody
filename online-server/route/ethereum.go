package route

import (
	"online-server/handler"
	"online-server/middleware"

	"github.com/gin-gonic/gin"
)

// EthereumRoutes 设置以太坊相关路由（保持向下兼容）
func EthereumRoutes(r *gin.Engine) {
	// 以太坊API路由组
	ethereum := r.Group("/api/ethereum")
	{
		// 公开访问的API
		ethereum.GET("/balance/:address", handler.GetBalance) // 查询以太坊余额

		// 需要身份验证的API
		authenticated := ethereum.Group("/")
		authenticated.Use(middleware.JWTAuth())
		{
			// 交易相关API，需要警员或管理员权限
			officer := authenticated.Group("/tx")
			officer.Use(middleware.OfficerRequired())
			{
				officer.POST("/prepare", handler.PrepareTransaction)       // 准备交易
				officer.POST("/sign-send", handler.SignAndSendTransaction) // 签名并发送交易
			}
		}
	}
}
