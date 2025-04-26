package main

import (
	"log"

	"web-se/config"
	"web-se/controllers"
	"web-se/middleware"

	"web-se/clog"

	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化日志系统

	if err = clog.Init(clog.DefaultConfig()); err != nil {
		log.Fatalf("初始化日志系统失败: %v", err)
	}
	clog.SetDefaultLevel(clog.DebugLevel)
	defer clog.Sync() // 刷新缓冲区

	clog.Info("系统启动")
	clog.Info("配置加载成功",
		clog.String("port", cfg.Port),
		clog.Bool("debug", cfg.Debug),
		clog.String("log_file", cfg.LogFile),
		clog.String("log_dir", cfg.LogDir),
	)

	// 设置Gin模式
	if cfg.Debug {
		gin.SetMode(gin.DebugMode)
		clog.Debug("Gin设置为调试模式")
	} else {
		gin.SetMode(gin.ReleaseMode)
		clog.Info("Gin设置为生产模式")
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
	clog.Info("服务器启动", clog.String("port", cfg.Port))
	if err := r.Run(":" + cfg.Port); err != nil {
		clog.Fatal("启动服务器失败", clog.String("error", err.Error()))
	}
}
