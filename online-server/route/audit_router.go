package route

import (
	"online-server/handler"
	"online-server/middleware"

	"github.com/gin-gonic/gin"
)

func AuditRoutes(r *gin.Engine) {
	group := r.Group("/api/audit-logs")
	group.Use(middleware.JWTAuth(), middleware.AuditorRequired())
	{
		group.GET("", handler.ListAuditLogs)
		group.GET("/", handler.ListAuditLogs)
		group.GET("/export", handler.ExportAuditLogs)
		group.GET("/:id", handler.GetAuditLog)
	}
}
