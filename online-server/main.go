package main

import (
	"log"
	"online-server/route"
	"online-server/utils"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 加载环境变量
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("加载环境变量失败: %v", err)
	}
	// 初始化数据库
	err = utils.InitDB()
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer utils.CloseDB()

	// 创建 Gin 路由器（没有默认中间件）
	r := gin.New()

	// 设置路由和中间件
	route.Setup(r)

	// 启动服务器
	log.Println("服务器启动在 :8080 端口")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
