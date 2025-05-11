package routes

import (
	"online-server/handlers"

	"online-server/utils"

	"github.com/gin-gonic/gin"
)

func UserRoutes(r *gin.Engine) {
	// 公开路由（无需认证）
	public := r.Group("/api")
	{
		public.POST("/login", handlers.Login)
		public.POST("/register", handlers.Register)
	}

	// 用户路由（需认证）
	users := r.Group("/api/users")
	users.Use(utils.JWTAuth())
	{
		// 用户管理 - 所有认证用户
		users.GET("/profile", handlers.GetCurrentUser)          // 获取当前登录用户信息
		users.POST("/logout", handlers.Logout)                  // 退出登录
		users.POST("/change-password", handlers.ChangePassword) // 修改密码

		// 用户列表 - 管理员功能
		admin := users.Group("/")
		admin.Use(utils.AdminRequired())
		{
			admin.GET("/", handlers.GetUsers)               // 获取所有用户
			admin.GET("/:id", handlers.GetUserByID)         // 获取指定用户信息
		}
	}
}
