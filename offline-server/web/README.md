# Web HTTP API 服务模块

## 概述

本模块提供了一个基于 RESTful API 的 HTTP 服务，是加密资产托管系统的核心接口层。系统包含以下核心功能：

1. 用户认证与授权管理
2. 密钥生成任务创建与管理
3. 交易签名任务创建与管理
4. 密钥分享管理
5. 角色权限控制

## 架构

```
+------------+    +------------+    +------------+
| 客户端应用 |    | 管理终端   |    | 其他系统   |
+-----+------+    +-----+------+    +-----+------+
      |                 |                 |
      +--------+--------+--------+--------+
               |
        +------+------+
        | HTTP API    |
        | 服务        |
        +------+------+
               |
      +--------+--------+--------+
      |                 |        |
+-----+------+   +------+------+ |
| 数据存储    |   | WebSocket  | |
| 服务        |   | 服务       | |
+------------+    +------------+ |
                                 |
                        +--------+------+
                        | 其他微服务    |
                        +---------------+
```

## 安装与运行

### 依赖

- Go 1.16+
- `github.com/gin-gonic/gin`
- `github.com/golang-jwt/jwt`
- GORM 库及 SQLite 驱动

### 启动服务

```go
// 启动服务并支持优雅关闭
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

web.RunWithGracefulShutdown(ctx, 8080)
```

## API 接口

### 用户认证 API

#### 1. 用户注册

```
POST /user/register

请求体:
{
  "username": "用户名",
  "password": "密码",
  "email": "电子邮件"
}

响应:
{
  "message": "注册成功",
  "user": {
    "id": 1,
    "username": "用户名",
    "email": "电子邮件",
    "role": "guest"
  }
}
```

#### 2. 用户登录

```
POST /user/login

请求体:
{
  "username": "用户名",
  "password": "密码"
}

响应:
{
  "token": "JWT令牌",
  "user": {
    "id": 1,
    "username": "用户名",
    "email": "电子邮件",
    "role": "admin"
  }
}
```

#### 3. 验证认证状态

```
POST /user/checkAuth

请求头:
Authorization: JWT令牌

响应:
{
  "status": "认证有效"
}
```

#### 4. 用户登出

```
POST /user/logout

请求头:
Authorization: JWT令牌

响应:
{
  "message": "登出成功"
}
```

### 用户管理 API (仅限管理员)

#### 1. 获取用户列表

```
GET /user/admin/users

请求头:
Authorization: JWT令牌

响应:
{
  "code": 200,
  "users": [
    {
      "id": 1,
      "username": "admin",
      "email": "admin@example.com",
      "role": "admin"
    },
    {
      "id": 2,
      "username": "user1",
      "email": "user1@example.com",
      "role": "participant"
    }
  ]
}
```

#### 2. 更新用户角色

```
PUT /user/admin/users/:id/role

请求头:
Authorization: JWT令牌

请求体:
{
  "role": "coordinator"
}

响应:
{
  "code": 200,
  "msg": "用户角色更新成功"
}
```

### 密钥管理 API

#### 1. 创建密钥生成任务

```
POST /key/generate

请求头:
Authorization: JWT令牌

请求体:
{
  "threshold": 2,
  "participants": ["user1", "user2", "user3"]
}

响应:
{
  "code": 200,
  "key_id": "key_20250416150405_1234",
  "status": "invited"
}
```

#### 2. 创建签名任务

```
POST /key/sign

请求头:
Authorization: JWT令牌

请求体:
{
  "key_id": "key_20250416150405_1234",
  "data": "需要签名的数据",
  "participants": ["user1", "user2"],
  "account_addr": "0x1234...5678"
}

响应:
{
  "code": 200,
  "sign_id": "key_20250416150405_1234",
  "key_id": "key_20250416150405_1234",
  "status": "invited"
}
```

#### 3. 获取任务状态

```
GET /key/status/:id

请求头:
Authorization: JWT令牌

响应:
{
  "code": 200,
  "id": "key_20250416150405_1234",
  "type": "keygen",
  "status": "completed",
  "detail": {
    "key_id": "key_20250416150405_1234",
    "initiator": "admin",
    "threshold": 2,
    "participants": ["user1", "user2", "user3"],
    "status": "completed",
    "created_at": "2025-04-16T15:04:05Z"
  }
}
```

### 密钥分享 API

#### 1. 获取用户所有密钥分享

```
GET /share

请求头:
Authorization: JWT令牌

响应:
{
  "code": 200,
  "shares": [
    {
      "key_id": "key_20250416150405_1234",
      "user_id": "1",
      "account_addr": "0x1234...5678",
      "share_data": "加密的分享数据"
    }
  ]
}
```

#### 2. 获取用户特定密钥分享

```
GET /share/:keyID

请求头:
Authorization: JWT令牌

响应:
{
  "code": 200,
  "share": {
    "key_id": "key_20250416150405_1234",
    "user_id": "1",
    "account_addr": "0x1234...5678",
    "share_data": "加密的分享数据"
  }
}
```

## 权限控制

系统定义了四种用户角色，每种角色具有不同的权限：

1. **Admin (管理员)**
   - 可以管理所有用户（查看、修改角色）
   - 可以创建密钥生成和签名任务
   - 可以查看所有任务状态

2. **Coordinator (协调员)**
   - 可以创建密钥生成和签名任务
   - 可以查看自己参与的任务状态

3. **Participant (参与者)**
   - 可以参与密钥生成和签名任务
   - 可以查看自己参与的任务

4. **Guest (访客)**
   - 基本访问权限
   - 无法执行敏感操作

## 中间件

### 1. 认证中间件 (AuthMiddleware)

验证用户身份并设置上下文信息：

```go
// 使用方式
adminGroup := userGroup.Group("/admin")
adminGroup.Use(AuthMiddleware())
{
    adminGroup.GET("/users", handler.ListUsers)
    // ...其他需要认证的路由
}
```

### 2. 密钥操作权限中间件 (KeyAuthMiddleware)

验证用户是否具有密钥操作权限（仅 Coordinator 和 Admin 角色）：

```go
// 使用方式
keyGroup := r.Group("/key")
keyGroup.Use(KeyAuthMiddleware())
{
    keyGroup.POST("/generate", handler.GenerateKey)
    // ...其他密钥操作路由
}
```

### 3. CORS 中间件 (CorsMiddleware)

处理跨域资源共享：

```go
// 使用方式
r := gin.Default()
r.Use(CorsMiddleware())
```

## 代码文件说明

### `router.go`

- 定义和初始化所有 API 路由
- 设置中间件
- 配置认证和权限控制

### `web.go`

- 包含 HTTP 服务器的启动和关闭逻辑
- 支持优雅关闭功能
- 初始化数据库连接

### `handler/handler.go`

- 定义所有 API 的请求处理函数
- 处理请求参数验证
- 调用服务层处理业务逻辑
- 格式化 API 响应

### `service/service.go`

- 实现业务逻辑
- 处理用户认证逻辑
- 与数据库交互

## 与其他模块的集成

### 1. 与 WebSocket 服务集成

Web API 服务负责创建密钥生成和签名任务，然后由 WebSocket 服务负责任务的实时执行和通信。

### 2. 与存储服务集成

Web API 服务通过存储接口与数据库进行交互，保存用户信息、密钥分享和任务状态。

## 安全注意事项

1. **身份验证**: 使用 JWT 进行用户身份验证，确保令牌安全性
2. **权限控制**: 严格按照用户角色分配权限，避免越权访问
3. **数据验证**: 对所有用户输入进行严格验证，防止注入攻击
4. **HTTPS**: 在生产环境中使用 HTTPS 保护通信安全
5. **日志安全**: 确保日志中不包含敏感信息

## 错误处理

系统统一使用 JSON 格式返回错误信息：

```json
{
  "error": "错误信息描述"
}
```

常见 HTTP 状态码使用:
- 200: 请求成功
- 400: 请求参数错误
- 401: 未认证或认证失败
- 403: 权限不足
- 404: 资源不存在
- 500: 服务器内部错误