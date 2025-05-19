package route

import (
	"online-server/handler"
	"online-server/middleware"

	"github.com/gin-gonic/gin"
)

func TransactionRouter(r *gin.Engine) {
	// 交易相关路由
	transactionGroup := r.Group("/api/transaction")
	{
		// 公开API
		transactionGroup.GET("/balance/:address", handler.GetBalance) // 获取账户余额

		// 需要警员或管理员权限的API
		officer := transactionGroup.Group("/tx")
		officer.Use(middleware.JWTAuth(), middleware.OfficerRequired())
		{
			officer.POST("/prepare", handler.PrepareTransaction)       // 准备交易
			officer.POST("/sign-send", handler.SignAndSendTransaction) // 签名并发起交易
		}
	}
}
