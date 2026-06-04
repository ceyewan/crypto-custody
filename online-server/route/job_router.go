package route

import (
	"online-server/handler"
	"online-server/middleware"

	"github.com/gin-gonic/gin"
)

func JobRoutes(r *gin.Engine) {
	jobs := r.Group("/api/jobs")
	jobs.Use(middleware.JWTAuth())
	{
		jobs.GET("", handler.ListJobs)
		jobs.GET("/", handler.ListJobs)
		jobs.GET("/:id", handler.GetJob)
		jobs.GET("/:id/result", handler.JobResult)
	}
}
