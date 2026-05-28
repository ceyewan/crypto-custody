package main

import (
	"log"
	"online-server/route"
	"online-server/utils"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// 加载环境变量
	err := godotenv.Load()
	if err != nil {
		log.Printf("未加载 .env 文件，将使用系统环境变量: %v", err)
	}

	jwtSecret := strings.TrimSpace(os.Getenv("JWT_SECRET_KEY"))
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
	bindAddr := getEnv("BACKEND_BIND_ADDR", "0.0.0.0")
	port := getEnv("BACKEND_PORT", getEnv("PORT", "22221"))
	addr := bindAddr + ":" + port
	if bindAddr == "" || bindAddr == "0.0.0.0" {
		addr = ":" + port
	}
	log.Printf("服务器启动在 %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
