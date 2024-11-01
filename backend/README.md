# 在线系统后端架构

## 目录结构

```
.
├── cmd
│   └── server
│       └── main.go	# 程序入口
├── config
│   ├── config.g    # 配置结构定义
│   └── config.yaml	# 配置文件
├── internal
│   ├── api         # API 层
│   │   ├── dto
│   │   │   ├── case.go
│   │   │   └── user.go
│   │   ├── handler	# 请求处理层
│   │   │   ├── case_handler.go
│   │   │   ├── handler.go
│   │   │   ├── log_handler.go
│   │   │   ├── user_handler.go
│   │   │   └── wallet_handler.go
│   │   ├── middleware     # 中间件
│   │   │   ├── auth.go    
│   │   │   ├── logger.go
│   │   │   └── recover.go
│   │   ├── router # 路由配置
│   │   │   ├── API Documentation.md
│   │   │   └── router.go
│   │   └── server.go
│   ├── model # 数据模型
│   │   ├── case.go
│   │   ├── log.go
│   │   ├── user.go
│   │   └── wallet.go
│   ├── service # 业务逻辑层
│   │   ├── case_service.go
│   │   ├── log_service.go
│   │   ├── service.go
│   │   ├── user_service.go
│   │   └── wallet_service.go
│   └── pkg # 内部通用包
│       ├── auth # 认证相关
│       ├── blockchain # 区块链相关
│       │   └── blockchain.go
│       ├── db
│       │   └── db.go
│       └── utils
│ 
├── migrations # 数据库迁移文件
├── pkg # 可能被外部使用的包
├── Dockerfile
├── go.mod
└── go.sum
```

