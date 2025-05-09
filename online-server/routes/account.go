package routes

import (
	"backend/handlers"

	"backend/utils"

	"github.com/gin-gonic/gin"
)

func AccountRoutes(r *gin.Engine) {
	accounts := r.Group("/api/accounts")
	{
		accounts.Use(utils.JWTAuth())
		{
			accounts.GET("/", handlers.GetAccounts)
			accounts.POST("/", handlers.CreateAccount)
			accounts.GET("/transferAll", handlers.TransferAll)
			accounts.GET("/updateBalance", handlers.UpdateBalance)
			accounts.POST("/packTransferData", handlers.PackTransferData)
			accounts.POST("/sendTransaction", handlers.SendTransaction)
		}

	}
}
