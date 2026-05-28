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
	CaseRoutes(r)        // 案件相关路由
	OfflineTaskRoutes(r) // 离线任务相关路由
	AuditRoutes(r)       // 审计日志路由
	BackupRoutes(r)      // 备份恢复路由
	TestDataRoutes(r)    // 测试数据和批量任务路由
}
