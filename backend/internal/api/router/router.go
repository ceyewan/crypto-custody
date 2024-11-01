// internal/api/router/router.go
package router

import (
	"crypto-custody/internal/api/handler"
	"crypto-custody/internal/api/middleware"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine, h *handler.Handler) {
	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	// API v1
	v1 := r.Group("/api/v1")

	// 认证相关路由
	auth := v1.Group("/auth")
	{
		auth.POST("/register", h.User.Register)
		auth.POST("/login", h.User.Login)
		auth.POST("/refresh", h.User.Refresh)
	}

	// 需要认证的路由
	authorized := v1.Group("/")
	authorized.Use(middleware.Auth())
	{
		// 用户相关
		users := authorized.Group("/users")
		{
			users.GET("/me", h.User.GetProfile)
			users.PUT("/me", h.User.UpdateProfile)
		}

		// 管理员路由
		admin := authorized.Group("/admin")
		admin.Use(middleware.RequireRole("SYSTEM_ADMIN"))
		{
			admin.GET("/users", h.User.List)
			admin.POST("/users", h.User.Create)
			admin.PUT("/users/:id/roles", h.User.UpdateRoles)
			admin.DELETE("/users/:id", h.User.Delete)
		}

		// 案件相关
		cases := authorized.Group("/cases")
		{
			cases.POST("/", middleware.RequireRole("SYSTEM_ADMIN"), h.Case.Create)
			cases.GET("/", h.Case.List)
			cases.GET("/:id", h.Case.Get)
			cases.PUT("/:id", middleware.RequireRoleOrPermission("SYSTEM_ADMIN", "CASE_WRITE"), h.Case.Update)
			cases.DELETE("/:id", middleware.RequireRole("SYSTEM_ADMIN"), h.Case.Delete)
			cases.POST("/:id/permissions", middleware.RequireRole("SYSTEM_ADMIN"), h.Case.AssignPermissions)
			cases.DELETE("/:id/permissions/:userId", middleware.RequireRole("SYSTEM_ADMIN"), h.Case.RevokePermissions)
		}

		// 钱包相关
		wallets := authorized.Group("/wallets")
		{
			wallets.POST("/", middleware.RequireRoleOrPermission("SYSTEM_ADMIN", "CASE_WRITE"), h.Wallet.Create)
			wallets.GET("/", h.Wallet.List)
			wallets.GET("/:address", h.Wallet.Get)
			wallets.DELETE("/:address", middleware.RequireRole("SYSTEM_ADMIN"), h.Wallet.Delete)
			wallets.POST("/:address/transfer", middleware.RequireRoleOrPermission("SYSTEM_ADMIN", "CASE_WRITE"), h.Wallet.Transfer)
		}

		// 日志相关
		logs := authorized.Group("/logs")
		{
			logs.GET("/system", h.Log.ListSystemLogs)
			logs.GET("/cases", h.Log.ListCaseLogs)
			logs.GET("/transactions", h.Log.ListTransactionLogs)
			logs.DELETE("/:id", middleware.RequireRole("SYSTEM_ADMIN"), h.Log.Delete)
		}
	}
}
