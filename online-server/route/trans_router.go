package route

import (
	"online-server/handler"
	"online-server/middleware"

	"github.com/gin-gonic/gin"
)

func TransactionRouter(r *gin.Engine) {
	// 交易相关路由
	transactionGroup := r.Group("/transaction")
	{
		authenticated := transactionGroup.Group("/")
		authenticated.Use(middleware.JWTAuth())
		{
			officer := authenticated.Group("/tx")
			{
				officer.GET("/balance/:address", handler.GetBalance)       // 获取账户余额
				officer.POST("/prepare", handler.PrepareTransaction)       // 准备交易
				officer.POST("/sign-send", handler.SignAndSendTransaction) // 签名并发起交易
			}
		}
	}
}
