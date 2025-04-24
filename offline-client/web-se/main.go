package main

import (
	"log"

	"web-se/config"
	"web-se/controllers"
	"web-se/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 设置Gin模式
	if cfg.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建Gin引擎
	r := gin.Default()

	// 注册中间件
	r.Use(middleware.ErrorHandler())

	// 注册API路由
	api := r.Group("/api")
	{
		v1 := api.Group("/v1")
		{
			// MPC操作相关路由
			mpc := v1.Group("/mpc")
			{
				mpc.POST("/keygen", controllers.KeyGeneration)
				mpc.POST("/sign", controllers.SignMessage)
			}
		}
	}

	// 启动服务器
	log.Printf("服务器启动在 %s 端口", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
}
