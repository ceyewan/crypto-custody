package route

import (
	"online-server/handler"
	"online-server/middleware"

	"github.com/gin-gonic/gin"
)

// Setup 设置并注册所有路由
func Setup(r *gin.Engine) {
	// 设置全局中间件
	middleware.Setup(r)

	// API 路由组
	api := r.Group("/api")
	{
		// 公开路由
		api.POST("/login", handler.Login)
		api.POST("/register", handler.Register)
		api.POST("/check-auth", handler.CheckAuth) // 验证Token是否有效

		// 以太坊公开API
		api.GET("/ethereum/balance/:address", handler.GetBalance) // 查询以太坊余额

		// 交易公开API
		api.GET("/accounts/transactions/:id", handler.GetTransaction)                // 获取交易详情
		api.GET("/accounts/transactions/hash/:hash", handler.GetTransactionByHash)   // 通过哈希获取交易
		api.GET("/accounts/transactions/user/:address", handler.GetUserTransactions) // 获取用户交易历史

		// ===== 需要认证的路由 =====
		authenticated := api.Group("")
		authenticated.Use(middleware.JWTAuth())
		{
			// 用户相关API
			users := authenticated.Group("/users")
			{
				// 所有认证用户可访问
				users.GET("/profile", handler.GetCurrentUser)          // 获取当前登录用户信息
				users.POST("/logout", handler.Logout)                  // 退出登录
				users.POST("/change-password", handler.ChangePassword) // 修改密码

				// 管理员功能
				admin := users.Group("/admin")
				admin.Use(middleware.AdminRequired())
				{
					admin.GET("/users", handler.GetUsers)                  // 获取所有用户
					admin.GET("/users/:id", handler.GetUserByID)           // 获取指定用户信息
					admin.PUT("/users/:id/role", handler.UpdateUserRole)   // 更新用户角色
					admin.PUT("/users/:id/username", handler.UpdateUserID) // 更新用户名
					admin.DELETE("/users/:id", handler.DeleteUser)         // 删除用户
				}
			}

			// 账户相关API
			accounts := authenticated.Group("/accounts")
			{
				// 查询账户
				accounts.GET("", handler.GetUserAccounts)    // 获取用户的账户列表
				accounts.GET("/all", handler.GetAllAccounts) // 获取所有账户列表（仅管理员）
				accounts.GET("/:id", handler.GetAccountByID) // 获取账户详情

				// 账户操作
				accounts.POST("", handler.CreateAccount)                     // 创建账户
				accounts.PUT("/:id", handler.UpdateAccount)                  // 更新账户
				accounts.PATCH("/:id/balance", handler.UpdateAccountBalance) // 更新账户余额
				accounts.DELETE("/:id", handler.DeleteAccount)               // 删除账户
				accounts.POST("/batch-import", handler.BatchImportAccounts)  // 批量导入账户
			}

			// 以太坊相关API
			ethereum := authenticated.Group("/ethereum")
			{
				// 交易相关API，需要警员或管理员权限
				tx := ethereum.Group("/tx")
				tx.Use(middleware.OfficerRequired())
				{
					tx.POST("/prepare", handler.PrepareTransaction)       // 准备交易
					tx.POST("/sign-send", handler.SignAndSendTransaction) // 签名并发送交易
				}
			}
		}
	}
}
