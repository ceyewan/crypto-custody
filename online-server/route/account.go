package route

import (
	"online-server/handler"
	"online-server/utils"

	"github.com/gin-gonic/gin"
)

func AccountRoutes(r *gin.Engine) {
	// 账户相关API路由组
	accounts := r.Group("/api/accounts")
	{
		// 交易相关API
		transactions := accounts.Group("/transactions")
		{
			// 公开访问的API
			transactions.GET("/:id", handler.GetTransaction)                // 获取交易详情
			transactions.GET("/hash/:hash", handler.GetTransactionByHash)   // 通过哈希获取交易
			transactions.GET("/user/:address", handler.GetUserTransactions) // 获取用户交易历史
		}

		// 需要认证的接口可以在后续添加
		authenticated := accounts.Group("/")
		authenticated.Use(utils.JWTAuth())
		{
			// 添加需要认证的路由
		}
	}
}
