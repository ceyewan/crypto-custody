# Web模块开发文档

## 简介

Web模块是加密资产托管系统的核心接口层，提供了基于RESTful风格的HTTP服务，实现了用户认证、密钥生成、交易签名等核心功能。该模块采用分层架构，包括路由层、处理器层和服务层，确保了系统的可维护性和扩展性。

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

## 主要功能

1. **用户管理**：注册、登录、身份验证和角色管理
2. **密钥生成**：创建和管理密钥生成任务
3. **交易签名**：发起和管理分布式签名流程
4. **基于角色的权限控制**：细粒度权限管理

## 角色与权限

系统定义了四种用户角色，每种角色拥有不同的权限：

1. **Admin (管理员)**
   - 管理所有用户（查看列表、修改角色）
   - 创建和管理密钥生成任务
   - 创建和管理签名任务
   - 访问所有密钥分享信息

2. **Coordinator (协调员)**
   - 创建和管理密钥生成任务
   - 创建和管理签名任务
   - 查看与其相关的任务状态

3. **Participant (参与者)**
   - 参与密钥生成和签名任务
   - 查看自己参与的任务状态

4. **Guest (访客)**
   - 具有最基本的系统访问权限
   - 新用户的默认角色

## 开始使用

### 启动服务

#### 基本启动方式

```go
import "offline-server/web"

// 启动服务，监听8080端口
web.Run(8080)
```

#### 支持优雅关闭的启动方式

```go
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

## 用户管理功能

### 1. 用户注册

注册新用户账号，默认角色为Guest。

**请求**:
- **URL**: `/user/register`
- **方法**: `POST`
- **内容类型**: `application/json`
- **请求体**:
  ```json
  {
    "username": "新用户名",
    "password": "密码",
    "email": "用户邮箱"
  }
  ```

**响应**:
```json
{
  "message": "注册成功",
  "user": {
    "id": 1,
    "username": "新用户名",
    "email": "用户邮箱",
    "role": "guest"
  }
}
```

### 2. 用户登录

验证用户凭据并返回JWT令牌，用于后续API请求的认证。

**请求**:
- **URL**: `/user/login`
- **方法**: `POST`
- **内容类型**: `application/json`
- **请求体**:
  ```json
  {
    "username": "用户名",
    "password": "密码"
  }
  ```

**响应**:
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "username": "用户名",
    "email": "用户邮箱",
    "role": "admin"
  }
}
```

### 3. 验证认证状态

验证JWT令牌是否有效。

**请求**:
- **URL**: `/user/checkAuth`
- **方法**: `POST`
- **请求头**: `Authorization: <JWT令牌>`

**响应**:
```json
{
  "status": "认证有效",
  "user": "用户名"
}
```

### 4. 用户登出

使当前用户的JWT令牌失效。

**请求**:
- **URL**: `/user/logout`
- **方法**: `POST`
- **请求头**: `Authorization: <JWT令牌>`

**响应**:
```json
{
  "message": "登出成功"
}
```

### 5. 获取用户列表（仅管理员）

获取系统中所有用户的列表。

**请求**:
- **URL**: `/user/admin/users`
- **方法**: `GET`
- **请求头**: `Authorization: <JWT令牌>`

**响应**:
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

### 6. 更新用户角色（仅管理员）

更新指定用户的角色。

**请求**:
- **URL**: `/user/admin/users/:id/role`
- **方法**: `PUT`
- **请求头**: `Authorization: <JWT令牌>`
- **请求体**:
  ```json
  {
    "role": "coordinator"
  }
  ```

**响应**:
```json
{
  "code": 200,
  "msg": "用户角色更新成功"
}
```

## 密钥生成功能

### 1. 创建密钥生成会话密钥

生成一个唯一的会话密钥，用于后续的WebSocket通信。

**请求**:
- **URL**: `/keygen/create/:initiator`
- **方法**: `GET`
- **请求头**: `Authorization: <JWT令牌>`
- **URL参数**: 
  - `initiator`: 发起者用户名

**响应**:
```json
{
  "session_key": "keygen_20230419150405_username"
}
```

### 2. 密钥生成流程

创建会话密钥后，密钥生成任务通过WebSocket通信进行。完整的流程为：

1. 调用Web API获取会话密钥
2. 通过WebSocket建立连接
3. 发送用户身份注册消息
4. 发送密钥生成请求
5. 处理密钥生成响应
6. 接收密钥生成结果

**示例代码**:

```javascript
// 1. 获取会话密钥
async function getSessionKey() {
  const response = await fetch(`/keygen/create/${username}`, {
    method: 'GET',
    headers: { 'Authorization': token }
  });
  const data = await response.json();
  return data.session_key;
}

// 2. 建立WebSocket连接
const sessionKey = await getSessionKey();
const ws = new WebSocket('ws://server-address/ws');

// 3. 发送用户身份注册
ws.onopen = () => {
  ws.send(JSON.stringify({
    type: "register",
    username: username,
    role: "coordinator", 
    token: token
  }));
};

// 4. 发送密钥生成请求
function sendKeyGenRequest() {
  ws.send(JSON.stringify({
    type: "keygen_request",
    session_key: sessionKey,
    threshold: 3,
    total_parts: 5,
    participants: ["user1", "user2", "user3", "user4", "user5"]
  }));
}

// 5. 处理WebSocket消息
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  
  switch(message.type) {
    case "register_complete":
      if (message.success) {
        sendKeyGenRequest();
      }
      break;
    case "keygen_complete":
      if (message.success) {
        console.log("密钥生成成功，地址:", message.address);
      } else {
        console.error("密钥生成失败:", message.message);
      }
      break;
    case "error":
      console.error("错误:", message.message, "详情:", message.details);
      break;
  }
};
```

## 签名功能

### 1. 创建签名会话密钥

生成一个唯一的签名会话密钥，用于后续的WebSocket通信。

**请求**:
- **URL**: `/sign/create/:initiator`
- **方法**: `GET`
- **请求头**: `Authorization: <JWT令牌>`
- **URL参数**: 
  - `initiator`: 发起者用户名

**响应**:
```json
{
  "session_key": "sign_20230419160405_username"
}
```

### 2. 签名流程

创建会话密钥后，签名任务通过WebSocket通信进行。完整的流程为：

1. 调用Web API获取会话密钥
2. 通过WebSocket建立连接
3. 发送用户身份注册消息
4. 发送签名请求
5. 处理签名响应
6. 接收签名结果

**示例代码**:

```javascript
// 1. 获取会话密钥
async function getSignSessionKey() {
  const response = await fetch(`/sign/create/${username}`, {
    method: 'GET',
    headers: { 'Authorization': token }
  });
  const data = await response.json();
  return data.session_key;
}

// 2. 建立WebSocket连接
const sessionKey = await getSignSessionKey();
const ws = new WebSocket('ws://server-address/ws');

// 3. 发送用户身份注册
ws.onopen = () => {
  ws.send(JSON.stringify({
    type: "register",
    username: username,
    role: "coordinator", 
    token: token
  }));
};

// 4. 发送签名请求
function sendSignRequest() {
  ws.send(JSON.stringify({
    type: "sign_request",
    session_key: sessionKey,
    threshold: 3,
    total_parts: 5,
    data: "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef", // 要签名的数据
    address: "0x1234...5678", // 账户地址
    participants: ["user1", "user2", "user3"]
  }));
}

// 5. 处理WebSocket消息
ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  
  switch(message.type) {
    case "register_complete":
      if (message.success) {
        sendSignRequest();
      }
      break;
    case "sign_complete":
      if (message.success) {
        console.log("签名成功，结果:", message.signature);
      } else {
        console.error("签名失败:", message.message);
      }
      break;
    case "error":
      console.error("错误:", message.message, "详情:", message.details);
      break;
  }
};
```

## 完整使用流程

### 1. 用户管理流程

1. **用户注册**：新用户通过 `/user/register` 接口创建账号
2. **登录获取令牌**：用户使用 `/user/login` 登录并获取JWT令牌
3. **角色管理**：管理员通过 `/user/admin/users/:id/role` 更新用户角色

### 2. 密钥生成流程

1. **创建会话**：协调者通过 `/keygen/create/:initiator` 获取会话密钥
2. **WebSocket通信**：
   - 建立WebSocket连接
   - 发送注册消息
   - 发送密钥生成请求
   - 协调者等待所有参与者完成操作
   - 接收密钥生成完成通知

### 3. 签名流程

1. **创建会话**：协调者通过 `/sign/create/:initiator` 获取会话密钥
2. **WebSocket通信**：
   - 建立WebSocket连接
   - 发送注册消息
   - 发送签名请求
   - 协调者等待所有参与者完成操作
   - 接收签名完成通知

## 错误处理

所有API响应遵循一致的JSON格式：

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

- **200**: 请求成功
- **400**: 请求参数错误
- **401**: 未认证或认证失败
- **403**: 权限不足
- **404**: 资源不存在
- **500**: 服务器内部错误

## 安全最佳实践

1. **使用HTTPS**: 生产环境中必须使用HTTPS确保传输安全
2. **JWT令牌管理**: 
   - 为令牌设置合理的过期时间
   - 存储在安全位置，如HttpOnly Cookie
   - 不在客户端存储敏感信息
3. **定期更新密码**: 鼓励用户定期更新密码
4. **角色权限验证**: 所有敏感操作必须验证用户角色
5. **输入验证**: 服务器端对所有输入进行验证

## 常见问题与解决方案

### 1. JWT令牌过期

**问题**: 用户令牌过期导致API请求失败。

**解决方案**: 
- 客户端检测到401错误时自动引导用户重新登录
- 实现令牌刷新机制

```javascript
async function callAPI(url, method, data) {
  try {
    const response = await fetch(url, {
      method: method,
      headers: { 'Authorization': getToken() },
      body: data ? JSON.stringify(data) : undefined
    });
    
    if (response.status === 401) {
      // 令牌过期，重定向到登录页面
      window.location.href = "/login";
      return;
    }
    
    return await response.json();
  } catch (error) {
    console.error("API调用错误:", error);
  }
}
```

### 2. 密钥生成参与者不在线

**问题**: 密钥生成过程中部分参与者不在线。

**解决方案**:
- 设置超时机制
- 提供重试功能
- 允许重新选择参与者

### 3. 签名过程中的错误处理

**问题**: 签名过程中可能发生错误。

**解决方案**:
- 实现错误恢复机制
- 提供详细的错误信息
- 记录完整的操作日志

## 与其他模块的集成

Web模块与系统的其他组件紧密集成：

1. **与WebSocket模块集成**: 
   - Web API负责创建会话密钥
   - WebSocket服务负责实时任务执行

2. **与存储模块集成**:
   - 用户数据管理
   - 密钥分片存储
   - 任务状态管理

## 扩展与自定义

Web API设计支持扩展和自定义：

1. **添加新的API端点**:
   ```go
   // 在router.go中添加新路由
   func initCustomRouter(r *gin.Engine) {
       customGroup := r.Group("/custom")
       customGroup.Use(AuthMiddleware())
       {
           customGroup.GET("/endpoint", handler.CustomHandler)
       }
   }
   
   // 在Register()函数中注册新路由
   func Register() *gin.Engine {
       // ... 现有代码 ...
       initCustomRouter(r)
       // ... 现有代码 ...
   }
   ```

2. **自定义中间件**:
   ```go
   // 创建自定义中间件
   func CustomMiddleware() gin.HandlerFunc {
       return func(c *gin.Context) {
           // 中间件逻辑
           c.Next()
       }
   }
   
   // 应用中间件
   customGroup.Use(CustomMiddleware())
   ```

## 总结

Web模块为加密资产托管系统提供了完整的用户和任务管理API。通过简单清晰的接口，使客户端应用能够轻松地与系统集成，实现用户管理、密钥生成和交易签名等核心功能。模块的分层设计确保了系统的可维护性和可扩展性，满足了不同规模部署的需求。 