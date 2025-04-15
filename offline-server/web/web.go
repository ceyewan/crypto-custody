package web

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"offline-server/web/db"
	"offline-server/web/model"
	"time"
)

// Run 启动Web服务器
func Run(port int) {
	// 初始化数据库
	if err := db.Init(); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 自动迁移模型
	if err := db.AutoMigrate(&model.User{}); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// 创建Gin引擎
	router := Register()

	// 启动HTTP服务器
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// 在goroutine中启动服务器
	go func() {
		log.Printf("服务器启动，监听端口 %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()
}

// RunWithGracefulShutdown 启动Web服务器并支持优雅关闭
func RunWithGracefulShutdown(ctx context.Context, port int) {
	// 初始化数据库
	if err := db.Init(); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 自动迁移模型
	if err := db.AutoMigrate(&model.User{}); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	// 创建Gin引擎
	router := Register()

	// 启动HTTP服务器
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// 在goroutine中启动服务器
	go func() {
		log.Printf("服务器启动，监听端口 %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("服务器错误: %v", err)
		}
	}()

	// 等待上下文取消信号
	<-ctx.Done()
	log.Println("正在关闭Web服务器...")

	// 设置5秒的超时时间来优雅关闭
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Web服务器关闭错误: %v", err)
	} else {
		log.Println("Web服务器已安全关闭")
	}
}
