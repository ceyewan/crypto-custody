package main

import (
	"log"
	"online-server/ethereum"
	"online-server/routes"
	"online-server/servers"
	"online-server/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化数据库
	utils.InitDB()
	defer utils.CloseDB()

	// 初始化以太坊服务
	ethService, err := ethereum.GetInstance()
	if err != nil {
		log.Printf("警告: 以太坊服务初始化失败: %v", err)
	} else {
		defer ethService.Close()
	}

	// 初始化以太坊交易服务
	err = servers.InitEthService()
	if err != nil {
		log.Printf("警告: 以太坊交易服务初始化失败: %v", err)
	}

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
	r.Run(":8080")
}
