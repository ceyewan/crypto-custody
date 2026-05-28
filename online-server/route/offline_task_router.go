package route

import (
	"online-server/handler"
	"online-server/middleware"

	"github.com/gin-gonic/gin"
)

func OfflineTaskRoutes(r *gin.Engine) {
	group := r.Group("/api/offline-tasks")
	group.Use(middleware.JWTAuth())
	{
		group.GET("", handler.ListOfflineTasks)
		group.GET("/", handler.ListOfflineTasks)
		group.GET("/:id", handler.GetOfflineTask)

		write := group.Group("")
		write.Use(middleware.OfficerRequired())
		{
			write.POST("/custody-keygen", handler.CreateCustodyKeygenTask)
			write.GET("/:id/export", handler.ExportOfflineTask)
			write.POST("/:id/import-result", handler.ImportOfflineTaskResult)
		}
	}
}
