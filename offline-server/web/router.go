package web

import (
	"net/http"
	"offline-server/storage/model"
	"offline-server/tools"
	"offline-server/web/handler"
	"offline-server/web/service"

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
	initOfflineRouter(r)

	// 处理404请求
	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "error": "404 Not Found"})
	})

	return r
}

func initOfflineRouter(r *gin.Engine) {
	offlineGroup := r.Group("/offline")
	offlineGroup.Use(AuthMiddleware())
	{
		offlineGroup.GET("/tasks/:task_no", handler.GetOfflineTask)
		offlineGroup.GET("/keys/:offline_key_id", handler.GetOfflineKey)
		offlineGroup.GET("/shards/mine", handler.ListMyKeyShards)
		offlineGroup.GET("/participation/mine", handler.ListMyParticipationRecords)
	}

	offlineAdminGroup := r.Group("/offline")
	offlineAdminGroup.Use(AdminAuthMiddleware())
	{
		offlineAdminGroup.POST("/tasks/import", handler.ImportOfflineTask)
		offlineAdminGroup.POST("/tasks/:task_no/keygen/start", handler.BuildKeygenTaskRequest)
		offlineAdminGroup.POST("/tasks/:task_no/sign/start", handler.BuildSignTaskRequest)
		offlineAdminGroup.GET("/results/:task_no/download", handler.DownloadOfflineResult)
		offlineAdminGroup.POST("/shards/:shard_id/transfer", handler.TransferKeyShard)
		offlineAdminGroup.POST("/keys/:offline_key_id/destroy", handler.DestroyOfflineKey)
		offlineAdminGroup.GET("/backup/download", handler.DownloadBackup)
		offlineAdminGroup.POST("/backups/hot", handler.CreateHotBackup)
		offlineAdminGroup.POST("/backups/cold/export", handler.CreateColdBackup)
		offlineAdminGroup.GET("/backups", handler.ListBackups)
		offlineAdminGroup.GET("/backups/:id/download", handler.DownloadBackupRecord)
		offlineAdminGroup.POST("/backups/:id/restore", handler.RestoreBackup)
		offlineAdminGroup.POST("/backups/:id/verify", handler.VerifyBackup)
	}

	offlineAuditGroup := r.Group("/offline")
	offlineAuditGroup.Use(AuditAuthMiddleware())
	{
		offlineAuditGroup.GET("/tasks", handler.ListOfflineTasks)
		offlineAuditGroup.GET("/keys", handler.ListOfflineKeys)
		offlineAuditGroup.GET("/shards", handler.ListKeyShards)
		offlineAuditGroup.GET("/audit", handler.ListAuditLogs)
		offlineAuditGroup.GET("/approvals", handler.ListApprovals)
	}
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
	adminGroup.Use(AdminAuthMiddleware())
	{
		adminGroup.GET("/users", handler.ListUsers)               // 获取用户列表
		adminGroup.PUT("/users/:id/role", handler.UpdateUserRole) // 更新用户角色
		adminGroup.PUT("/users/:id/status", handler.UpdateUserStatus)
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
		seGroup.GET("/list", handler.ListSe)      // 获取安全芯片列表
		seGroup.POST("/create", handler.CreateSe) // 创建安全芯片记录
		seGroup.DELETE("/:se_id", handler.DeleteSe)
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
		if !ensureActiveUser(c, userName) {
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
		if !ensureActiveUser(c, userName) {
			return
		}

		// 只有管理员可以发起 keygen/sign，管理员本人也可以作为参与方。
		if role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "权限不足，需要管理员角色"})
			return
		}

		// 设置用户信息到上下文
		c.Set("userName", userName)
		c.Set("role", role)
		c.Next()
	}
}

// AuditAuthMiddleware 审计查询权限验证中间件。
func AuditAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "未提供认证令牌"})
			return
		}

		userName, role, err := tools.ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "无效的认证令牌"})
			return
		}
		if !ensureActiveUser(c, userName) {
			return
		}

		if role != "admin" && role != "auditor" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "权限不足，需要管理员或审计员角色"})
			return
		}

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
		if !ensureActiveUser(c, userName) {
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

func ensureActiveUser(c *gin.Context, userName string) bool {
	user, err := service.GetUserByUserName(userName)
	if err != nil || user.Status != model.UserStatusActive {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "账号已停用或不存在"})
		return false
	}
	return true
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
