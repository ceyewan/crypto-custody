package main

import (
	"log"
	"online-server/routes"
	"online-server/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化数据库
	err := utils.InitDB()
	if err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	defer utils.CloseDB()

	// 创建 Gin 路由
	r := gin.Default()

	// 添加 CORS 中间件
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// 注册路由
	routes.UserRoutes(r)
	routes.AccountRoutes(r)
	routes.EthereumRoutes(r)

	// 启动服务器
	log.Println("服务器启动在 :8080 端口")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
