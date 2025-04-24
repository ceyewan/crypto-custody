package main

import (
	"log"

	"web-se/config"
	"web-se/controllers"
	"web-se/middleware"
	"web-se/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化日志系统
	logger, err := utils.InitLogger(cfg)
	if err != nil {
		log.Fatalf("初始化日志系统失败: %v", err)
	}
	defer logger.Sync() // 刷新缓冲区

	utils.LogInfo("系统启动")
	utils.LogInfo("配置加载成功",
		utils.String("port", cfg.Port),
		utils.Bool("debug", cfg.Debug),
		utils.String("log_file", cfg.LogFile))

	// 设置Gin模式
	if cfg.Debug {
		gin.SetMode(gin.DebugMode)
		utils.LogDebug("Gin设置为调试模式")
	} else {
		gin.SetMode(gin.ReleaseMode)
		utils.LogInfo("Gin设置为生产模式")
	}

	// 创建Gin引擎
	r := gin.Default()

	// 注册中间件
	r.Use(middleware.ErrorHandler())
	r.Use(middleware.LoggerMiddleware())

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
	utils.LogInfo("服务器启动", utils.String("port", cfg.Port))
	if err := r.Run(":" + cfg.Port); err != nil {
		utils.LogFatal("启动服务器失败", utils.Error(err))
	}
}
