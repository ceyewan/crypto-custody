package route

import (
	"online-server/middleware"

	"github.com/gin-gonic/gin"
)

// Setup 设置并注册所有路由
func Setup(r *gin.Engine) {
	// 设置全局中间件
	middleware.Setup(r)

	// 注册各个模块的路由
	UserRoutes(r)        // 用户相关路由
	AccountRoutes(r)     // 账户相关路由
	TransactionRouter(r) // 交易相关路由
}
