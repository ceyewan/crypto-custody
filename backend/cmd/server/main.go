package main

import (
	"crypto-custody/config"
	"crypto-custody/internal/api"
	"crypto-custody/internal/pkg/db"
	"crypto-custody/internal/service"
	"log"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化数据库连接
	if err := db.InitDB(cfg.Database); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 初始化服务
	services := service.NewServices(db.GetDB())

	// 启动HTTP服务器
	server := api.NewServer(cfg, services)
	if err := server.Run(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
