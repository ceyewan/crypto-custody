package route

import (
	"online-server/handler"
	"online-server/middleware"

	"github.com/gin-gonic/gin"
)

// AccountRoutes 设置账户相关路由
func AccountRoutes(r *gin.Engine) {
	// 账户相关API路由组
	accounts := r.Group("/api/accounts")
	{
		// 公开API
		accounts.GET("/address/:address", handler.GetAccountByAddress) // 根据账户地址查询账户信息

		// 需要认证的API
		authenticated := accounts.Group("")
		authenticated.Use(middleware.JWTAuth())
		{
			authenticated.GET("", handler.ListAccounts)
			authenticated.GET("/", handler.ListAccounts)
			authenticated.GET("/template", handler.AccountTemplate)
			authenticated.GET("/export", handler.ExportAccounts)

			officer := authenticated.Group("/officer")
			officer.Use(middleware.OfficerRequired())
			{
				officer.GET("/", handler.GetUserAccounts)            // 获取用户的账户信息
				officer.POST("/create", handler.CreateAccount)       // 创建账户
				officer.POST("/import", handler.BatchImportAccounts) // 批量导入账户
			}
			authenticated.POST("", middleware.OfficerRequired(), handler.CreateAccount)
			authenticated.POST("/", middleware.OfficerRequired(), handler.CreateAccount)
			authenticated.POST("/import", middleware.OfficerRequired(), handler.BatchImportAccounts)
			authenticated.POST("/:id/sync-balance", middleware.OfficerRequired(), handler.SyncAccountBalance)
			authenticated.POST("/sync-balances", middleware.OfficerRequired(), handler.BatchSyncAccountBalances)

			// 管理员专用API
			admin := authenticated.Group("/admin")
			admin.Use(middleware.AdminRequired())
			{
				admin.GET("/all", handler.GetAllAccounts)   // 获取所有账户信息（仅管理员）
				admin.DELETE("/:id", handler.DeleteAccount) // 删除账户（仅管理员）
			}
		}
	}
}
