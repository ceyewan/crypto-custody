package middleware

import (
	"net/http"
	"online-server/model"
	"online-server/service"
	"online-server/utils"
	"time"

	"github.com/ceyewan/clog"
	"github.com/gin-gonic/gin"
)

// Setup 设置全局中间件
func Setup(r *gin.Engine) {
	// 注册日志中间件
	r.Use(LoggerMiddleware())
	// 注册恢复中间件
	r.Use(RecoveryMiddleware())
	// 注册CORS中间件
	r.Use(CORSMiddleware())
}

// CORSMiddleware CORS中间件
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

// JWTAuth JWT认证中间件
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := clog.Module("middleware")

		authorization := c.Request.Header.Get("Authorization")
		if authorization == "" {
			logger.Warn("访问需要认证的API但未提供令牌",
				clog.String("path", c.Request.URL.Path),
				clog.String("ip", c.ClientIP()))
			utils.ResponseWithError(c, http.StatusUnauthorized, "需要登录")
			c.Abort()
			return
		}

		userName, role, err := utils.ValidateToken(authorization)
		if err != nil {
			logger.Warn("令牌验证失败",
				clog.Err(err),
				clog.String("token_prefix", authorization[:10]+"..."),
				clog.String("ip", c.ClientIP()),
				clog.String("path", c.Request.URL.Path))
			utils.ResponseWithError(c, http.StatusUnauthorized, "令牌无效")
			c.Abort()
			return
		}

		logger.Info("用户认证成功",
			clog.String("username", userName),
			clog.String("role", role),
			clog.String("path", c.Request.URL.Path))

		// 设置用户信息到上下文
		c.Set("Username", userName)
		c.Set("Role", role)

		// 获取用户信息并设置到上下文
		userService, err := service.GetUserServiceInstance()
		if err == nil {
			user, err := userService.GetUserByUsername(userName)
			if err == nil {
				// 设置完整的用户模型到上下文中
				c.Set("user", user)
			}
		}

		c.Next()
	}
}

// AdminRequired 管理员权限检查中间件
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := clog.Module("middleware")

		role, exists := c.Get("Role")
		if !exists {
			logger.Warn("访问管理员资源但未提供认证信息",
				clog.String("path", c.Request.URL.Path),
				clog.String("ip", c.ClientIP()))
			utils.ResponseWithError(c, http.StatusUnauthorized, utils.ErrorUnauthorized)
			c.Abort()
			return
		}

		// 检查是否为管理员
		if role.(string) != string(model.RoleAdmin) {
			logger.Warn("非管理员尝试访问管理员资源",
				clog.String("username", c.GetString("Username")),
				clog.String("role", role.(string)),
				clog.String("path", c.Request.URL.Path),
				clog.String("ip", c.ClientIP()))
			utils.ResponseWithError(c, http.StatusForbidden, utils.ErrorForbidden+", 需要管理员权限")
			c.Abort()
			return
		}

		logger.Info("管理员权限验证通过",
			clog.String("username", c.GetString("Username")),
			clog.String("path", c.Request.URL.Path))
		c.Next()
	}
}

// OfficerRequired 警员权限检查中间件
func OfficerRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := clog.Module("middleware")

		role, exists := c.Get("Role")
		if !exists {
			logger.Warn("访问警员资源但未提供认证信息",
				clog.String("path", c.Request.URL.Path),
				clog.String("ip", c.ClientIP()))
			utils.ResponseWithError(c, http.StatusUnauthorized, utils.ErrorUnauthorized)
			c.Abort()
			return
		}

		// 检查是否为管理员或警员
		if role.(string) != string(model.RoleAdmin) && role.(string) != string(model.RoleOfficer) {
			logger.Warn("游客尝试访问警员资源",
				clog.String("username", c.GetString("Username")),
				clog.String("role", role.(string)),
				clog.String("path", c.Request.URL.Path),
				clog.String("ip", c.ClientIP()))
			utils.ResponseWithError(c, http.StatusForbidden, utils.ErrorForbidden+", 需要警员或管理员权限")
			c.Abort()
			return
		}

		logger.Info("警员权限验证通过",
			clog.String("username", c.GetString("Username")),
			clog.String("role", role.(string)),
			clog.String("path", c.Request.URL.Path))
		c.Next()
	}
}

// LoggerMiddleware 日志中间件
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		logger := clog.Module("http")

		// 开始时间
		startTime := time.Now()

		// 处理请求
		c.Next()

		// 执行时间
		latency := time.Since(startTime)

		// 获取用户信息
		username, exists := c.Get("Username")
		userStr := "未登录"
		if exists {
			userStr = username.(string)
		}

		// 记录日志
		if c.Writer.Status() >= 400 {
			// 错误请求日志
			logger.Warn("API请求异常",
				clog.String("method", c.Request.Method),
				clog.String("path", c.Request.URL.Path),
				clog.Int("status", c.Writer.Status()),
				clog.String("ip", c.ClientIP()),
				clog.Duration("latency", latency),
				clog.String("user", userStr),
				clog.String("user_agent", c.Request.UserAgent()),
			)
		} else {
			// 正常请求日志
			logger.Info("API请求",
				clog.String("method", c.Request.Method),
				clog.String("path", c.Request.URL.Path),
				clog.Int("status", c.Writer.Status()),
				clog.String("ip", c.ClientIP()),
				clog.Duration("latency", latency),
				clog.String("user", userStr),
			)
		}
	}
}

// RecoveryMiddleware 恢复中间件，防止服务崩溃
func RecoveryMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger := clog.Module("recovery")
				logger.Error("服务发生严重异常",
					clog.Any("error", err),
					clog.String("path", c.Request.URL.Path),
					clog.String("method", c.Request.Method),
					clog.String("ip", c.ClientIP()),
					clog.String("user_agent", c.Request.UserAgent()),
				)

				// 获取用户信息（如果有）
				if username, exists := c.Get("Username"); exists {
					logger.Error("异常用户信息",
						clog.String("username", username.(string)),
						clog.String("path", c.Request.URL.Path))
				}

				utils.ResponseWithError(c, http.StatusInternalServerError, utils.ErrorInternalServerError)
			}
		}()
		c.Next()
	}
}
