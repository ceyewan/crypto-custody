package route

import (
	"online-server/handler"
	"online-server/middleware"

	"github.com/gin-gonic/gin"
)

// AccountRoutes 设置账户相关路由（保持向下兼容）
func AccountRoutes(r *gin.Engine) {
	// 账户相关API路由组
	accounts := r.Group("/api/accounts")
	{
		// 交易相关API - 分离到单独的路由组
		transactions := accounts.Group("/transactions")
		{
			// 公开访问的API
			transactions.GET("/:id", handler.GetTransaction)                // 获取交易详情
			transactions.GET("/hash/:hash", handler.GetTransactionByHash)   // 通过哈希获取交易
			transactions.GET("/user/:address", handler.GetUserTransactions) // 获取用户交易历史
		}

		// 账户管理API - 需要认证
		authenticated := accounts.Group("/")
		authenticated.Use(middleware.JWTAuth())
		{
			// 查询账户
			authenticated.GET("", handler.GetUserAccounts)    // 获取用户的账户列表
			authenticated.GET("/all", handler.GetAllAccounts) // 获取所有账户列表（仅管理员）
			authenticated.GET("/:id", handler.GetAccountByID) // 获取账户详情

			// 账户操作
			authenticated.POST("", handler.CreateAccount)                     // 创建账户
			authenticated.PUT("/:id", handler.UpdateAccount)                  // 更新账户
			authenticated.PATCH("/:id/balance", handler.UpdateAccountBalance) // 更新账户余额
			authenticated.DELETE("/:id", handler.DeleteAccount)               // 删除账户
			authenticated.POST("/batch-import", handler.BatchImportAccounts)  // 批量导入账户
		}
	}
}
