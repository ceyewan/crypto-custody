package routes

import (
	"online-server/handlers"
	"online-server/utils"

	"github.com/gin-gonic/gin"
)

func AccountRoutes(r *gin.Engine) {
	// 基础路由组
	accounts := r.Group("/api/accounts")
	
	// 公开的账户查询接口
	accounts.GET("/:address", handlers.GetAccountBalance) // 查询指定地址余额
	
	// 需要认证的接口
	auth := accounts.Group("/")
	auth.Use(utils.JWTAuth())
	{
		// 所有认证用户可访问
		auth.GET("/", handlers.GetAccounts) // 获取账户列表
		
		// 警员和管理员可访问
		officer := auth.Group("/")
		officer.Use(utils.OfficerRequired())
		{
			officer.POST("/", handlers.CreateAccount)            // 创建账户
			officer.POST("/packTransferData", handlers.PackTransferData) // 打包交易数据
			officer.POST("/sendTransaction", handlers.SubmitTransaction) // 提交交易
		}
		
		// 仅管理员可访问
		admin := auth.Group("/admin")
		admin.Use(utils.AdminRequired())
		{
			admin.GET("/transferAll", handlers.TransferAll)      // 批量转账
			admin.GET("/updateBalance", handlers.UpdateBalance)  // 更新所有账户余额
		}
	}
}
