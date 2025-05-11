package routes

import (
	"online-server/handlers"
	"online-server/utils"

	"github.com/gin-gonic/gin"
)

func EthereumRoutes(r *gin.Engine) {
	ethereum := r.Group("/api/ethereum")
	{
		// 公开访问的API
		ethereum.GET("/balance/:address", handlers.GetAccountBalance)
		ethereum.GET("/transaction/:id", handlers.GetTransactionStatus)
		ethereum.GET("/transactions/:address", handlers.GetUserTransactions)

		// 需要身份验证的API
		authenticated := ethereum.Group("/")
		authenticated.Use(utils.JWTAuth())
		{
			// 警员和管理员可访问的交易API
			officer := authenticated.Group("/")
			officer.Use(utils.OfficerRequired())
			{
				officer.POST("/prepare", handlers.PrepareTransaction)
				officer.POST("/sign", handlers.SignTransaction)
				officer.POST("/send/:id", handlers.ProcessTransaction)
			}
			
			// 仅管理员可访问的API
			admin := authenticated.Group("/admin")
			admin.Use(utils.AdminRequired())
			{
				admin.POST("/check-pending", handlers.CheckPendingTransactions)
			}
		}
	}
}