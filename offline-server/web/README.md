# 加密资产托管系统 Web API 服务

## 概述

本模块提供了基于 RESTful API 的 HTTP 服务，作为加密资产托管系统的核心接口层。系统包含以下核心功能：

1. 用户认证与权限管理
2. 密钥生成任务管理
3. 交易签名任务管理
4. 密钥分享管理
5. 基于角色的权限控制

## 系统架构

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

## 启动服务

### 基本启动方式

```go
// 导入包
import "offline-server/web"

// 启动服务
web.Run(8080)
```

### 支持优雅关闭的启动方式

```go
// 导入包
import (
    "context"
    "offline-server/web"
)

// 创建上下文
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// 启动服务并支持优雅关闭
web.RunWithGracefulShutdown(ctx, 8080)
```

## API 端点详情

### 用户管理 API

#### 1. 用户登录
- **URL**: `/user/login`
- **方法**: `POST`
- **描述**: 验证用户凭据并返回JWT令牌
- **请求体**:
  ```json
  {
    "username": "用户名",
    "password": "密码"
  }
  ```
- **响应**:
  ```json
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

#### 2. 用户注册
- **URL**: `/user/register`
- **方法**: `POST`
- **描述**: 创建新用户
- **请求体**:
  ```json
  {
    "username": "用户名",
    "password": "密码",
    "email": "电子邮件"
  }
  ```
- **响应**:
  ```json
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

#### 3. 验证认证状态
- **URL**: `/user/checkAuth`
- **方法**: `POST`
- **描述**: 验证JWT令牌是否有效
- **请求头**: `Authorization: <JWT令牌>`
- **响应**:
  ```json
  {
    "status": "认证有效"
  }
  ```

#### 4. 用户登出
- **URL**: `/user/logout`
- **方法**: `POST`
- **描述**: 使当前用户的JWT令牌失效
- **请求头**: `Authorization: <JWT令牌>`
- **响应**:
  ```json
  {
    "message": "登出成功"
  }
  ```

#### 5. 获取用户列表（需要管理员权限）
- **URL**: `/user/admin/users`
- **方法**: `GET`
- **描述**: 获取系统中所有用户的列表
- **请求头**: `Authorization: <JWT令牌>`
- **响应**:
  ```json
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

#### 6. 更新用户角色（需要管理员权限）
- **URL**: `/user/admin/users/:id/role`
- **方法**: `PUT`
- **描述**: 更新指定用户的角色
- **请求头**: `Authorization: <JWT令牌>`
- **请求体**:
  ```json
  {
    "role": "coordinator"
  }
  ```
- **响应**:
  ```json
  {
    "code": 200,
    "msg": "用户角色更新成功"
  }
  ```

### 密钥生成 API

#### 1. 创建密钥生成任务
- **URL**: `/keygen/create`
- **方法**: `POST`
- **描述**: 发起新的密钥生成任务
- **权限要求**: Coordinator 或 Admin 角色
- **请求头**: `Authorization: <JWT令牌>`
- **请求体**:
  ```json
  {
    "threshold": 2,
    "participants": ["user1", "user2", "user3"]
  }
  ```
- **响应**:
  ```json
  {
    "code": 200,
    "key_id": "keygen_20250419150405_1234",
    "status": "invited"
  }
  ```

#### 2. 获取密钥生成会话信息
- **URL**: `/keygen/session/:id`
- **方法**: `GET`
- **描述**: 获取特定密钥生成会话的详细信息
- **权限要求**: Coordinator 或 Admin 角色
- **请求头**: `Authorization: <JWT令牌>`
- **响应**:
  ```json
  {
    "code": 200,
    "session": {
      "id": "keygen_20250419150405_1234",
      "initiator": "admin",
      "threshold": 2,
      "participants": ["user1", "user2", "user3"],
      "status": "in_progress",
      "created_at": "2025-04-19T15:04:05Z"
    }
  }
  ```

#### 3. 获取密钥生成任务状态
- **URL**: `/keygen/status/:id`
- **方法**: `GET`
- **描述**: 获取密钥生成任务的当前状态
- **权限要求**: Coordinator 或 Admin 角色
- **请求头**: `Authorization: <JWT令牌>`
- **响应**:
  ```json
  {
    "code": 200,
    "id": "keygen_20250419150405_1234",
    "status": "completed",
    "detail": {
      "key_id": "keygen_20250419150405_1234",
      "initiator": "admin",
      "threshold": 2,
      "participants": ["user1", "user2", "user3"],
      "status": "completed",
      "created_at": "2025-04-19T15:04:05Z",
      "completed_at": "2025-04-19T15:10:05Z"
    }
  }
  ```

### 签名 API

#### 1. 创建签名任务
- **URL**: `/sign/create`
- **方法**: `POST`
- **描述**: 发起新的交易签名任务
- **权限要求**: Coordinator 或 Admin 角色
- **请求头**: `Authorization: <JWT令牌>`
- **请求体**:
  ```json
  {
    "key_id": "keygen_20250419150405_1234",
    "data": "需要签名的数据（通常是交易哈希）",
    "participants": ["user1", "user2"],
    "account_addr": "0x1234...5678"
  }
  ```
- **响应**:
  ```json
  {
    "code": 200,
    "sign_id": "sign_20250419160405_5678",
    "key_id": "keygen_20250419150405_1234",
    "status": "invited"
  }
  ```

#### 2. 获取签名会话信息
- **URL**: `/sign/session/:id`
- **方法**: `GET`
- **描述**: 获取特定签名会话的详细信息
- **权限要求**: Coordinator 或 Admin 角色
- **请求头**: `Authorization: <JWT令牌>`
- **响应**:
  ```json
  {
    "code": 200,
    "session": {
      "id": "sign_20250419160405_5678",
      "key_id": "keygen_20250419150405_1234",
      "initiator": "admin",
      "data": "0x...",
      "participants": ["user1", "user2"],
      "account_addr": "0x1234...5678",
      "status": "in_progress",
      "created_at": "2025-04-19T16:04:05Z"
    }
  }
  ```

#### 3. 获取签名任务状态
- **URL**: `/sign/status/:id`
- **方法**: `GET`
- **描述**: 获取签名任务的当前状态
- **权限要求**: Coordinator 或 Admin 角色
- **请求头**: `Authorization: <JWT令牌>`
- **响应**:
  ```json
  {
    "code": 200,
    "id": "sign_20250419160405_5678",
    "status": "completed",
    "signature": "0xabcd...ef12",
    "detail": {
      "sign_id": "sign_20250419160405_5678",
      "key_id": "keygen_20250419150405_1234",
      "initiator": "admin",
      "data": "0x...",
      "participants": ["user1", "user2"],
      "account_addr": "0x1234...5678",
      "status": "completed",
      "created_at": "2025-04-19T16:04:05Z",
      "completed_at": "2025-04-19T16:10:05Z"
    }
  }
  ```

#### 4. 通过账户地址查询签名会话
- **URL**: `/sign/account/:addr`
- **方法**: `GET`
- **描述**: 根据账户地址查询相关的签名会话
- **权限要求**: Coordinator 或 Admin 角色
- **请求头**: `Authorization: <JWT令牌>`
- **响应**:
  ```json
  {
    "code": 200,
    "sessions": [
      {
        "id": "sign_20250419160405_5678",
        "key_id": "keygen_20250419150405_1234",
        "initiator": "admin",
        "data": "0x...",
        "participants": ["user1", "user2"],
        "account_addr": "0x1234...5678",
        "status": "completed",
        "created_at": "2025-04-19T16:04:05Z",
        "completed_at": "2025-04-19T16:10:05Z"
      }
    ]
  }
  ```

#### 5. 获取账户参与者
- **URL**: `/sign/participants/:addr`
- **方法**: `GET`
- **描述**: 获取与特定账户相关的参与者
- **权限要求**: Coordinator 或 Admin 角色
- **请求头**: `Authorization: <JWT令牌>`
- **响应**:
  ```json
  {
    "code": 200,
    "account": "0x1234...5678",
    "participants": ["user1", "user2", "user3"],
    "threshold": 2
  }
  ```

### 分享管理 API

#### 1. 获取会话相关的密钥分享
- **URL**: `/share/:session`
- **方法**: `GET`
- **描述**: 获取特定会话的密钥分享
- **权限要求**: Admin 角色
- **请求头**: `Authorization: <JWT令牌>`
- **响应**:
  ```json
  {
    "code": 200,
    "shares": [
      {
        "key_id": "keygen_20250419150405_1234",
        "user_id": "1",
        "account_addr": "0x1234...5678",
        "share_data": "加密的分享数据"
      }
    ]
  }
  ```

#### 2. 通过账户地址获取分享
- **URL**: `/share/account/:addr`
- **方法**: `GET`
- **描述**: 通过账户地址获取相关的密钥分享
- **权限要求**: Admin 角色
- **请求头**: `Authorization: <JWT令牌>`
- **响应**:
  ```json
  {
    "code": 200,
    "account": "0x1234...5678",
    "shares": [
      {
        "key_id": "keygen_20250419150405_1234",
        "user_id": "1",
        "account_addr": "0x1234...5678",
        "share_data": "加密的分享数据"
      }
    ]
  }
  ```

## 角色与权限

系统定义了四种用户角色，每种角色拥有不同的权限：

1. **Admin (管理员)**
   - 可以管理所有用户（查看列表、修改角色）
   - 可以创建和管理密钥生成任务
   - 可以创建和管理签名任务
   - 可以访问所有密钥分享信息
   - 拥有系统的最高权限

2. **Coordinator (协调员)**
   - 可以创建和管理密钥生成任务
   - 可以创建和管理签名任务
   - 可以查看与其相关的任务状态
   - 无法访问密钥分享的原始数据

3. **Participant (参与者)**
   - 可以参与密钥生成和签名任务
   - 可以查看自己参与的任务状态
   - 通常是密钥托管的实际执行者

4. **Guest (访客)**
   - 具有最基本的系统访问权限
   - 无法执行任何敏感操作
   - 新注册用户的默认角色

## 中间件

### 1. AuthMiddleware
基本认证中间件，用于验证用户身份并设置上下文信息。

### 2. KeyAuthMiddleware
专用于密钥操作的权限验证，要求用户必须是 Coordinator 或 Admin 角色。

### 3. AdminAuthMiddleware
严格的管理员权限验证中间件，要求用户必须是 Admin 角色。

### 4. CorsMiddleware
处理跨域资源共享，使API可以从不同域的前端应用访问。

## 错误码和响应格式

所有API响应均遵循一致的JSON格式：

### 成功响应
```json
{
  "code": 200,
  "message": "操作成功",
  // 其他数据...
}
```

### 错误响应
```json
{
  "code": 400, // 或其他错误码
  "error": "错误描述"
}
```

### 常见HTTP状态码
- 200: 请求成功
- 400: 请求参数错误
- 401: 未认证或认证失败
- 403: 权限不足
- 404: 资源不存在
- 500: 服务器内部错误

## 与其他模块集成

### WebSocket服务集成
密钥生成和签名任务的实时执行通过WebSocket服务完成，Web API负责任务的创建和状态查询。

### 存储模块集成
Web服务通过存储接口与数据库交互，用于管理用户信息、密钥分享和任务状态。

## 安全注意事项

1. **JWT认证**: 所有受保护的API均通过JWT令牌进行验证
2. **基于角色的访问控制**: 严格按照用户角色控制权限
3. **数据验证**: 对所有用户输入进行严格验证
4. **HTTPS**: 生产环境强烈推荐使用HTTPS
5. **安全日志**: 系统记录关键操作，但不记录敏感信息