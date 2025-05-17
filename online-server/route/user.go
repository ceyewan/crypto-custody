package route

import (
	"online-server/handler"

	"online-server/utils"

	"github.com/gin-gonic/gin"
)

func UserRoutes(r *gin.Engine) {
	// 公开路由（无需认证）
	public := r.Group("/api")
	{
		public.POST("/login", handler.Login)
		public.POST("/register", handler.Register)
	}

	// 用户路由（需认证）
	users := r.Group("/api/users")
	users.Use(utils.JWTAuth())
	{
		// 用户管理 - 所有认证用户
		users.GET("/profile", handler.GetCurrentUser)          // 获取当前登录用户信息
		users.POST("/logout", handler.Logout)                  // 退出登录
		users.POST("/change-password", handler.ChangePassword) // 修改密码

		// 用户列表 - 管理员功能
		admin := users.Group("/admin")
		admin.Use(utils.AdminRequired())
		{
			admin.GET("/users", handler.GetUsers)                // 获取所有用户
			admin.GET("/users/:id", handler.GetUserByID)         // 获取指定用户信息
			admin.PUT("/users/:id/role", handler.UpdateUserRole) // 更新用户角色
			admin.DELETE("/users/:id", handler.DeleteUser)       // 删除用户
		}
	}
}
