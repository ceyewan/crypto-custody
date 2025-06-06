package route

import (
	"online-server/handler"
	"online-server/middleware"

	"github.com/gin-gonic/gin"
)

// UserRoutes 设置用户相关路由
//
// 配置所有与用户管理相关的API端点，包括公开路由和需要认证的路由
//
// 参数：
// - r：Gin引擎实例，用于注册路由
func UserRoutes(r *gin.Engine) {
	// 公开路由（无需认证）
	public := r.Group("/api")
	{
		public.POST("/login", handler.Login)          // 用户登录
		public.POST("/register", handler.Register)    // 用户注册
		public.POST("/check-auth", handler.CheckAuth) // 验证Token是否有效
	}

	// 用户路由（需认证）
	users := r.Group("/api/users")
	users.Use(middleware.JWTAuth())
	{
		// 用户管理 - 所有认证用户
		users.GET("/profile", handler.GetCurrentUser)          // 获取当前登录用户信息
		users.POST("/logout", handler.Logout)                  // 退出登录
		users.POST("/change-password", handler.ChangePassword) // 修改自己的密码

		// 用户列表 - 管理员功能
		admin := users.Group("/admin")
		admin.Use(middleware.AdminRequired())
		{
			admin.GET("/users", handler.GetUsers)                         // 获取所有用户
			admin.GET("/users/:id", handler.GetUserByID)                  // 获取指定用户信息
			admin.PUT("/users/:id/role", handler.UpdateUserRole)          // 更新用户角色
			admin.PUT("/users/:id/username", handler.UpdateUserID)        // 更新用户名
			admin.PUT("/users/:id/password", handler.AdminChangePassword) // 管理员修改用户密码
			admin.DELETE("/users/:id", handler.DeleteUser)                // 删除用户
		}
	}
}
