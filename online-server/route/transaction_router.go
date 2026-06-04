package route

import (
	"online-server/handler"
	"online-server/middleware"

	"github.com/gin-gonic/gin"
)

func TransactionRouter(r *gin.Engine) {
	transactions := r.Group("/api/transactions")
	transactions.Use(middleware.JWTAuth())
	{
		transactions.GET("", handler.ListTransactionsV2)
		transactions.GET("/", handler.ListTransactionsV2)
		transactions.GET("/:id", handler.GetTransactionV2)
		ops := transactions.Group("")
		ops.Use(middleware.OfficerRequired())
		{
			ops.POST("", handler.CreateTransactionPrepared)
			ops.POST("/", handler.CreateTransactionPrepared)
			ops.POST("/import", handler.BatchImportTransactions)
			ops.POST("/:id/prepare", handler.PrepareTransactionV2)
			ops.GET("/:id/export-sign-task", handler.ExportSignTask)
			ops.POST("/:id/import-signature", handler.ImportTransactionSignature)
			ops.POST("/:id/broadcast", handler.BroadcastTransactionV2)
			ops.POST("/:id/check-receipt", handler.CheckTransactionReceiptV2)
		}
		admin := transactions.Group("")
		admin.Use(middleware.AdminRequired())
		{
			admin.DELETE("/:id", handler.DeleteTransaction)
		}
	}

	// 交易相关路由
	transactionGroup := r.Group("/api/transaction")
	{
		// 公开API
		transactionGroup.GET("/balance/:address", handler.GetBalance) // 获取账户余额

		// 需要认证的API
		authenticated := transactionGroup.Group("/")
		authenticated.Use(middleware.JWTAuth())
		{
			// 获取交易详情 - 所有认证用户都可以访问
			authenticated.GET("/:id", handler.GetTransactionByID)

			// 需要警员或管理员权限的API
			officer := authenticated.Group("/")
			officer.Use(middleware.OfficerRequired())
			{
				officer.GET("/list", handler.GetTransactionList)   // 获取交易列表 (警员+)
				officer.GET("/stats", handler.GetTransactionStats) // 获取交易统计 (警员+)
			}

			// 交易操作 - 需要警员或管理员权限
			txOperations := authenticated.Group("/tx")
			txOperations.Use(middleware.OfficerRequired())
			{
				txOperations.POST("/prepare", handler.PrepareTransaction)       // 准备交易
				txOperations.POST("/sign-send", handler.SignAndSendTransaction) // 签名并发起交易
			}

			// 管理员专用API
			admin := authenticated.Group("/admin")
			admin.Use(middleware.AdminRequired())
			{
				admin.GET("/all", handler.GetAllTransactions)       // 获取所有交易 (管理员)
				admin.GET("/stats", handler.GetAllTransactionStats) // 获取所有交易统计 (管理员)
				admin.DELETE("/:id", handler.DeleteTransaction)     // 删除交易 (管理员)
			}
		}
	}
}
