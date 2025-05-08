package web

import (
	"net/http"
	"offline-server/tools"
	"offline-server/web/handler"

	"github.com/gin-gonic/gin"
)

// Register 初始化并返回配置好的Gin引擎实例
func Register() *gin.Engine {
	r := gin.Default()
	r.Use(CorsMiddleware())

	// 初始化各模块路由
	initUserRouter(r)
	initKeyGenRouter(r) // 修改为密钥生成专用路由
	initSignRouter(r)   // 新增签名专用路由
	initShareRouter(r)
	initPushRouter(r)
	initSeRouter(r) // 新增SE相关路由

	// 处理404请求
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "error": "404 Not Found"})
	})

	return r
}

// initUserRouter 初始化用户相关路由
func initUserRouter(r *gin.Engine) {
	userGroup := r.Group("/user")

	// 公共API - 无需认证
	userGroup.POST("/login", handler.Login)       // 用户登录
	userGroup.POST("/register", handler.Register) // 用户注册

	// 需要认证的API
	userGroup.POST("/checkAuth", handler.CheckAuth) // 验证令牌有效性
	userGroup.POST("/logout", handler.Logout)       // 用户登出

	// 管理员API - 需要admin权限
	adminGroup := userGroup.Group("/admin")
	adminGroup.Use(AuthMiddleware())
	{
		adminGroup.GET("/users", handler.ListUsers)               // 获取用户列表
		adminGroup.PUT("/users/:id/role", handler.UpdateUserRole) // 更新用户角色
	}
}

// initKeyGenRouter 初始化密钥生成相关路由
func initKeyGenRouter(r *gin.Engine) {
	keyGenGroup := r.Group("/keygen")
	keyGenGroup.Use(KeyAuthMiddleware()) // 使用专门的中间件验证密钥操作权限
	{
		keyGenGroup.GET("/create/:initiator", handler.CreateKenGenSessionKey) // 创建密钥生成会话
		keyGenGroup.GET("/users", handler.GetAvailableUsers)                  // 获取可参与密钥生成的用户列表
	}
}

// initSignRouter 初始化签名相关路由
func initSignRouter(r *gin.Engine) {
	signGroup := r.Group("/sign")
	signGroup.Use(KeyAuthMiddleware()) // 使用专门的中间件验证密钥操作权限
	{
		signGroup.GET("/create/:initiator", handler.CreateSignSessionKey) // 创建签名会话
		signGroup.GET("/users/:address", handler.GetUsersByAddress)       // 获取特定地址的用户列表
	}
}

// initShareRouter 初始化密钥分享相关路由
func initShareRouter(r *gin.Engine) {
	shareGroup := r.Group("/share")
	shareGroup.Use(AdminAuthMiddleware()) // 需要认证
	{

	}
}

// initPushRouter 初始化消息推送相关路由
func initPushRouter(r *gin.Engine) {
	pushGroup := r.Group("/push")
	// 需要认证的路由
	pushGroup.Use(AuthMiddleware())
	{
		// 保留推送路由，待实现
	}
}

// initSeRouter 初始化安全芯片相关路由
func initSeRouter(r *gin.Engine) {
	seGroup := r.Group("/se")
	seGroup.Use(AdminAuthMiddleware()) // 需要管理员权限
	{
		seGroup.POST("/create", handler.CreateSe) // 创建安全芯片记录
	}
}

// AuthMiddleware 基于JWT的权限验证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从Header获取token
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未提供认证令牌"})
			return
		}

		// 验证token - 直接使用裸token
		userName, role, err := tools.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效的认证令牌"})
			return
		}

		// 设置用户信息到上下文
		c.Set("userName", userName)
		c.Set("role", role)
		c.Next()
	}
}

// KeyAuthMiddleware 密钥操作权限验证中间件
func KeyAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从Header获取token
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未提供认证令牌"})
			return
		}

		// 验证token - 直接使用裸token
		userName, role, err := tools.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效的认证令牌"})
			return
		}

		// 检查是否为Coordinator或Admin角色
		if role != "coordinator" && role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "权限不足，需要Coordinator或Admin角色"})
			return
		}

		// 设置用户信息到上下文
		c.Set("userName", userName)
		c.Set("role", role)
		c.Next()
	}
}

// AdminAuthMiddleware 管理员权限验证中间件
func AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从Header获取token
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未提供认证令牌"})
			return
		}

		// 验证token - 直接使用裸token
		userName, role, err := tools.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效的认证令牌"})
			return
		}

		// 检查是否为Admin角色
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "权限不足，需要Admin角色"})
			return
		}

		// 设置用户信息到上下文
		c.Set("userName", userName)
		c.Set("role", role)
		c.Next()
	}
}

// CorsMiddleware 返回处理跨域资源共享(CORS)的中间件
func CorsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method

		// 添加CORS相关响应头
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
		c.Header("Access-Control-Allow-Methods", "GET, OPTIONS, POST, PUT, DELETE")
		c.Set("content-type", "application/json")

		// 对OPTIONS请求直接返回成功
		if method == "OPTIONS" {
			c.JSON(http.StatusOK, nil)
			c.Abort()
			return
		}

		c.Next()
	}
}
