package routes

import (
	"backend/handlers"

	"backend/utils"

	"github.com/gin-gonic/gin"
)

func UserRoutes(r *gin.Engine) {
	users := r.Group("/api/users")
	{
		//users.GET("/", handlers.GetAccounts)
		users.POST("/login", handlers.Login)
		users.POST("/register", handlers.Register)
		users.Use(utils.JWTAuth())
		{
			users.GET("/", handlers.GetUsers)
			//users.GET("/:id", handlers.GetUser)
			//users.PUT("/:id", handlers.UpdateUser)
			//users.DELETE("/:id", handlers.DeleteUser)
		}
	}
}
