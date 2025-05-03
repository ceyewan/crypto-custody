package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"offline-server/manager"
	"offline-server/storage/db"
	"offline-server/web"
	"offline-server/ws"
)

const (
	// 配置文件和数据存储的路径
	DataDir        = "./data"
	LogsDir        = "./logs"
	ManagerBinPath = "./bin/gg20_sm_manager"
	DBFilePath     = "./data/users.db"
)

func main() {
	// 命令行参数
	webPort := flag.Int("web-port", 8080, "Web服务器监听端口")
	wsPort := flag.Int("ws-port", 8081, "WebSocket服务器监听端口")
	flag.Parse()

	// 确保必要的目录存在
	ensureDirectories()

	// 捕获终止信号
	ctx, cancel := context.WithCancel(context.Background())
	setupSignalHandler(cancel)

	// 初始化数据库
	if err := db.Init(); err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}

	// 启动Manager服务
	log.Println("正在启动Manager服务...")
	managerExitCh := startManager(ctx)

	// 启动WebSocket服务器
	log.Println("正在启动WebSocket服务器...")
	wsServer := ws.NewServer("ws://localhost:8081")
	if err := wsServer.Start(); err != nil {
		log.Fatalf("启动WebSocket服务器失败: %v", err)
	}
	log.Printf("WebSocket服务器已启动，监听端口 %d", *wsPort)

	// 启动Web服务器（支持优雅关闭）
	log.Println("正在启动Web服务器...")
	go web.RunWithGracefulShutdown(ctx, *webPort)
	log.Printf("Web服务器已启动，监听端口 %d", *webPort)

	// 等待终止信号
	<-ctx.Done()
	log.Println("收到终止信号，正在关闭服务...")

	// 优雅关闭服务
	wsServer.Stop() // 关闭WebSocket服务器

	// 等待Manager退出
	select {
	case <-managerExitCh:
		log.Println("Manager已关闭")
	case <-time.After(5 * time.Second):
		log.Println("等待Manager关闭超时")
	}

	log.Println("系统已关闭")
}

// ensureDirectories 确保必要的目录存在
func ensureDirectories() {
	dirs := []string{
		DataDir,
		LogsDir,
		filepath.Dir(ManagerBinPath),
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

// startManager 启动Manager服务
func startManager(ctx context.Context) <-chan struct{} {
	exitCh := make(chan struct{})

	go func() {
		defer close(exitCh)

		// 检查Manager可执行文件是否存在
		if _, err := os.Stat(ManagerBinPath); os.IsNotExist(err) {
			log.Printf("警告: Manager可执行文件不存在: %s", ManagerBinPath)
			log.Printf("请确保将gg20_sm_manager放置在正确位置")
			return
		}

		// 配置Manager
		config := manager.Config{
			BinaryPath:   ManagerBinPath,
			LogDir:       LogsDir,
			RestartDelay: 3 * time.Second,
			AutoRestart:  true,
		}

		// 创建并启动Manager进程
		managerProcess := manager.New(config)
		if err := managerProcess.Start(); err != nil {
			log.Printf("启动Manager失败: %v", err)
			return
		}

		// 等待上下文取消
		<-ctx.Done()
		log.Println("正在关闭Manager...")
		managerProcess.Stop()
	}()

	return exitCh
}
