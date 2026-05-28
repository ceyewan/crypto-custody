package route

import (
	"online-server/handler"
	"online-server/middleware"

	"github.com/gin-gonic/gin"
)

func TestDataRoutes(r *gin.Engine) {
	group := r.Group("/api/test-data")
	group.Use(middleware.JWTAuth(), middleware.AdminRequired())
	{
		group.POST("/seed", handler.SeedTestData)
		group.POST("/clear", handler.ClearTestData)
		group.GET("/summary", handler.TestDataSummary)
		group.GET("/templates/accounts", handler.TestAccountTemplate)
		group.GET("/templates/transactions", handler.TestTransactionTemplate)
	}

	jobs := r.Group("/api/jobs")
	jobs.Use(middleware.JWTAuth())
	{
		jobs.GET("", handler.ListJobs)
		jobs.GET("/", handler.ListJobs)
		jobs.GET("/:id", handler.GetJob)
		jobs.GET("/:id/result", handler.JobResult)
	}
}
