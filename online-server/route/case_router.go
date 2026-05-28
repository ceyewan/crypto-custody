package route

import (
	"online-server/handler"
	"online-server/middleware"

	"github.com/gin-gonic/gin"
)

func CaseRoutes(r *gin.Engine) {
	group := r.Group("/api/cases")
	group.Use(middleware.JWTAuth())
	{
		group.GET("", handler.ListCases)
		group.GET("/", handler.ListCases)
		group.GET("/:id", handler.GetCase)
		group.GET("/:id/accounts", handler.GetCaseAccounts)

		write := group.Group("")
		write.Use(middleware.OfficerRequired())
		{
			write.POST("", handler.CreateCase)
			write.POST("/", handler.CreateCase)
			write.PUT("/:id", handler.UpdateCase)
			write.POST("/:id/accounts", handler.LinkCaseAccount)
			write.DELETE("/:id/accounts/:accountId", handler.UnlinkCaseAccount)
			write.POST("/:id/custody-wallet/import-result", handler.ImportCustodyWalletResult)
		}

		admin := group.Group("")
		admin.Use(middleware.AdminRequired())
		{
			admin.DELETE("/:id", handler.DeleteCase)
		}
	}
}
