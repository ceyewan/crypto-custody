package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"offline-server/clog"
	"offline-server/storage/db"
	"offline-server/web"
	"offline-server/ws"
)

const (
	// 配置文件和数据存储的路径
	DataDir       = "./data"
	LogsDir       = "./logs"
	ManagerBinDir = "./bin"
)

func main() {
	// 命令行参数
	webPort := flag.Int("web-port", 8080, "Web服务器监听端口")
	wsPort := flag.Int("ws-port", 8081, "WebSocket服务器监听端口")
	wsHost := flag.String("ws-host", "0.0.0.0", "WebSocket服务器监听地址")
	flag.Parse()

	// 确保必要的目录存在
	ensureDirectories()

	// 捕获终止信号
	ctx, cancel := context.WithCancel(context.Background())
	setupSignalHandler(cancel)

	// 初始化日志系统
	if err := clog.Init(clog.DefaultConfig()); err != nil {
		log.Fatalf("初始化日志系统失败: %v", err)
	}
	clog.SetDefaultLevel(clog.DebugLevel)
	defer clog.Sync() // 刷新缓冲区

	// 初始化数据库
	if err := db.Init(); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 启动WebSocket服务器
	log.Println("正在启动WebSocket服务器...")
	wsServer := ws.NewServer(fmt.Sprintf("%s:%d", *wsHost, *wsPort))
	if err := wsServer.Start(); err != nil {
		log.Fatalf("启动WebSocket服务器失败: %v", err)
	}
	log.Printf("WebSocket服务器已启动，监听地址 %s:%d", *wsHost, *wsPort)

	// 启动Web服务器（支持优雅关闭）
	log.Println("正在启动Web服务器...")
	go web.RunWithGracefulShutdown(ctx, *webPort)
	log.Printf("Web服务器已启动，监听端口 %d", *webPort)

	// 等待终止信号
	<-ctx.Done()
	log.Println("收到终止信号，正在关闭服务...")

	// 优雅关闭服务
	wsServer.Stop() // 关闭WebSocket服务器

	log.Println("系统已关闭")
}

// ensureDirectories 确保必要的目录存在
func ensureDirectories() {
	dirs := []string{
		DataDir,
		LogsDir,
		ManagerBinDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("创建目录 %s 失败: %v", dir, err)
		}
	}
}

// setupSignalHandler 设置信号处理器
func setupSignalHandler(cancel context.CancelFunc) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		log.Printf("收到信号: %s", sig)
		cancel()
	}()
}
