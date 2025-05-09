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

	"github.com/gin-contrib/cors"
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

	// 配置CORS中间件
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

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

	controllers.Shutdown() // 关闭所有控制器相关资源
	clog.Info("所有控制器资源已清理")

	// 设置5秒超时关闭服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		clog.Error("服务器强制关闭", clog.String("error", err.Error()))
	}

	clog.Info("服务器已优雅退出")
}
