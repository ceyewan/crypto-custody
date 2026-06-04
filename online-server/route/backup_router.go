package route

import (
	"online-server/handler"
	"online-server/middleware"

	"github.com/gin-gonic/gin"
)

func BackupRoutes(r *gin.Engine) {
	group := r.Group("/api/backups")
	group.Use(middleware.JWTAuth(), middleware.AdminRequired())
	{
		group.POST("/hot", handler.CreateHotBackup)
		group.POST("/cold/export", handler.CreateColdBackup)
		group.POST("/cold/import", handler.ImportColdBackup)
		group.GET("", handler.ListBackups)
		group.GET("/", handler.ListBackups)
		group.GET("/:id/download", handler.DownloadBackup)
		group.POST("/:id/restore", handler.RestoreBackup)
		group.POST("/:id/verify", handler.VerifyBackup)
	}
}
