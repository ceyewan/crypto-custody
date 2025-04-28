package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// 初始化控制器
	if err := controllers.Init(); err != nil {
		clog.Fatal("控制器初始化失败", clog.String("error", err.Error()))
	}
	// 注册退出时清理资源的函数
	defer controllers.Shutdown()

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
				mpc.GET("/cplc", controllers.GetCPLC)
				mpc.POST("/delete", controllers.DeleteMessage)
			}
		}
	}

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	// 在goroutine中启动服务器
	go func() {
		clog.Info("服务器启动", clog.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			clog.Fatal("启动服务器失败", clog.String("error", err.Error()))
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	// 监听 SIGINT (Ctrl+C)、SIGTERM (docker stop) 等信号
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	clog.Info("接收到关闭信号，开始优雅退出...")

	// 设置5秒超时关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		clog.Error("服务器强制关闭", clog.String("error", err.Error()))
	}

	clog.Info("服务器已优雅退出")
}
