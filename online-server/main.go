package main

import (
	"log"
	"online-server/route"
	"online-server/utils"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 加载环境变量
	err := godotenv.Load()
	if err != nil {
		log.Printf("未加载 .env 文件，将使用系统环境变量: %v", err)
	}

	jwtSecret := os.Getenv("JWT_SECRET_KEY")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET_KEY 未设置")
	}
	utils.SetJWTKey([]byte(jwtSecret))

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
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("服务器启动在 :%s 端口", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
