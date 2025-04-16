# WebSocket 分布式密钥管理模块

## 概述

本模块提供了一个基于 WebSocket 的分布式密钥生成和签名系统，支持多方安全计算 (MPC) 协议。系统包含以下核心功能：

1. 分布式密钥生成
2. 分布式签名
3. 客户端管理
4. 会话状态跟踪

## 架构

```
+------------+    +------------+    +------------+
| Coordinator|    | Participant|    | Participant|
+-----+------+    +-----+------+    +-----+------+
      |                 |                 |
      +--------+--------+--------+--------+
               |                 |
        +------+------+   +------+------+
        | WebSocket   |   | WebSocket   |
        | Server      |   | Manager     |
        +------+------+   +------+------+
               |                 |
        +------+------+   +------+------+
        | KeyGen      |   | Signing     |
        | Service     |   | Service     |
        +-------------+   +-------------+
```

## 安装与运行

### 依赖

- Go 1.16+
- `github.com/gorilla/websocket`

### 启动服务

```go
server := ws.NewServer()
if err := server.Start(8080); err != nil {
    log.Fatal(err)
}
defer server.Stop()
```

## 消息协议

### 消息认证

所有 WebSocket 消息都需要包含有效的 JWT Token 进行身份验证。Token 需要从 Web 服务登录后获取，并在每个 WebSocket 消息中提供。

### 基本消息结构

所有消息都遵循以下基本结构：

```json
{
  "type": "消息类型",
  "user_id": "用户ID",
  "token": "JWT令牌",
  "payload": { /* 消息载荷，根据消息类型不同而不同 */ }
}
```

### 消息类型与交互流程

#### 1. 连接与注册

##### 1.1 注册请求 (register)

客户端连接后首先需要发送注册请求：

```json
{
  "type": "register",
  "token": "JWT令牌",
  "payload": {
    "user_id": "用户ID",
    "role": "coordinator|participant" 
  }
}
```

##### 1.2 注册确认 (register_confirm)

服务器确认注册成功：

```json
{
  "type": "register_confirm",
  "user_id": "用户ID",
  "payload": {
    "status": "success"
  }
}
```

#### 2. 密钥生成流程

##### 2.1 密钥生成请求 (keygen_request)

协调方发起密钥生成请求：

```json
{
  "type": "keygen_request",
  "user_id": "协调方ID",
  "token": "JWT令牌",
  "payload": {
    "key_id": "密钥ID",
    "threshold": 2,
    "total_parts": 3,
    "participants": ["参与方1", "参与方2", "..."]
  }
}
```

##### 2.2 密钥生成邀请 (keygen_invite)

服务器向各参与方发送邀请：

```json
{
  "type": "keygen_invite",
  "user_id": "参与方ID",
  "payload": {
    "key_id": "密钥ID",
    "threshold": 2,
    "total_parts": 3,
    "part_index": 1,
    "participants": ["参与方1", "参与方2", "..."]
  }
}
```

##### 2.3 密钥生成响应 (keygen_response)

参与方对邀请的响应：

```json
{
  "type": "keygen_response",
  "user_id": "参与方ID",
  "token": "JWT令牌",
  "payload": {
    "key_id": "密钥ID",
    "part_index": 1,
    "response": true
  }
}
```

##### 2.4 密钥生成参数 (keygen_params)

服务器向参与方发送密钥生成所需参数：

```json
{
  "type": "keygen_params",
  "user_id": "参与方ID",
  "payload": {
    "key_id": "密钥ID",
    "threshold": 2,
    "total_parts": 3,
    "part_index": 1,
    "output_file": "输出文件名",
    "account_addr": "账户地址"
  }
}
```

##### 2.5 密钥生成完成 (keygen_complete)

参与方将生成的密钥分享发送给服务器：

```json
{
  "type": "keygen_complete",
  "user_id": "参与方ID",
  "token": "JWT令牌",
  "payload": {
    "key_id": "密钥ID",
    "part_index": 1,
    "account_addr": "账户地址",
    "share_json": "序列化的密钥分享JSON"
  }
}
```

##### 2.6 密钥生成确认 (keygen_confirm)

服务器向协调方确认密钥生成完成：

```json
{
  "type": "keygen_confirm",
  "user_id": "协调方ID",
  "payload": {
    "key_id": "密钥ID",
    "status": "success",
    "account_addr": "账户地址"
  }
}
```

#### 3. 签名流程

##### 3.1 签名请求 (sign_request)

协调方发起签名请求：

```json
{
  "type": "sign_request",
  "user_id": "协调方ID",
  "token": "JWT令牌",
  "payload": {
    "key_id": "密钥ID",
    "data": "要签名的数据",
    "account_addr": "账户地址",
    "participants": ["参与方1", "参与方2", "..."]
  }
}
```

##### 3.2 签名邀请 (sign_invite)

服务器向各参与方发送签名邀请：

```json
{
  "type": "sign_invite",
  "user_id": "参与方ID",
  "payload": {
    "key_id": "密钥ID",
    "data": "要签名的数据",
    "account_addr": "账户地址",
    "part_index": 1,
    "participants": ["参与方1", "参与方2", "..."]
  }
}
```

##### 3.3 签名响应 (sign_response)

参与方对签名邀请的响应：

```json
{
  "type": "sign_response",
  "user_id": "参与方ID",
  "token": "JWT令牌",
  "payload": {
    "key_id": "密钥ID",
    "part_index": 1,
    "response": true
  }
}
```

##### 3.4 签名参数 (sign_params)

服务器向参与方发送签名所需参数：

```json
{
  "type": "sign_params",
  "user_id": "参与方ID",
  "payload": {
    "key_id": "密钥ID",
    "data": "要签名的数据",
    "part_index": 1,
    "participants": "1,2,3",
    "share_json": "密钥分享的JSON字符串"
  }
}
```

##### 3.5 签名结果 (sign_result)

参与方将签名结果发送给服务器：

```json
{
  "type": "sign_result",
  "user_id": "参与方ID",
  "token": "JWT令牌",
  "payload": {
    "key_id": "密钥ID",
    "part_index": 1,
    "signature": "签名结果"
  }
}
```

##### 3.6 签名完成 (sign_complete)

服务器向协调方发送最终签名结果：

```json
{
  "type": "sign_complete",
  "user_id": "协调方ID",
  "payload": {
    "key_id": "密钥ID",
    "status": "success",
    "signature": "最终签名",
    "data": "签名的数据",
    "account_addr": "账户地址"
  }
}
```

#### 4. 错误消息 (error)

出现错误时服务器会发送错误消息：

```json
{
  "type": "error",
  "payload": {
    "error": "错误信息"
  }
}
```

### 完整交互流程图

#### 密钥生成流程

```
协调方                       服务器                      参与方
  |                           |                           |
  |--- keygen_request ------->|                           |
  |                           |--- keygen_invite -------->|
  |                           |<-- keygen_response ------|
  |                           |--- keygen_params -------->|
  |                           |<-- keygen_complete ------|
  |<-- keygen_confirm --------|                           |
  |                           |                           |
```

#### 签名流程

```
协调方                       服务器                      参与方
  |                           |                           |
  |--- sign_request --------->|                           |
  |                           |--- sign_invite ---------->|
  |                           |<-- sign_response ---------|
  |                           |--- sign_params ---------->|
  |                           |<-- sign_result ----------|
  |<-- sign_complete ---------|                           |
  |                           |                           |
```

## 代码文件说明

### `client.go`

- **Client**: 表示一个 WebSocket 客户端连接，负责处理与客户端的通信。
- **NewClient**: 创建并初始化一个新的客户端实例。
- **Listen**: 监听客户端消息，处理消息并支持注册。
- **SendMessage**: 向客户端发送消息，支持重试机制。
- **SendMessageToUser**: 向特定用户发送消息。

### `server.go`

- **Server**: 表示 WebSocket 服务器，管理连接和消息处理。
- **NewServer**: 创建并初始化服务器实例。
- **Start**: 启动服务器，包括外部进程和 HTTP 服务。
- **Stop**: 优雅地停止服务器，关闭所有连接。
- **handleConnection**: 处理新的 WebSocket 连接请求。

### `types.go`

- 定义了消息类型和消息结构。
- **Message**: 基本消息结构，包含类型、用户 ID、令牌和载荷。
- **RegisterPayload**: 注册消息的载荷。
- **KeyGenRequestPayload**: 密钥生成请求的载荷。
- **SignRequestPayload**: 签名请求的载荷。

### `handler.go`

- **MessageHandler**: 处理所有WebSocket消息的分发与处理。
- **HandleMessage**: 根据消息类型分发并处理接收到的消息。
- **handleRegister**: 处理客户端注册请求。

### `storage.go`

- **Storage**: 定义状态存储接口，提供客户端管理、会话管理功能。
- **PersistentStorage**: Storage接口的持久化实现。
- **KeyGenSession/SignSession**: 密钥生成会话和签名会话的数据结构。

### `keygen.go`

- 包含密钥生成相关的处理函数。
- **HandleKeyGenRequest**: 处理密钥生成请求。
- **HandleKeyGenResponse**: 处理密钥生成响应。
- **HandleKeyGenComplete**: 处理密钥生成完成通知。

### `signing.go`

- 包含签名相关的处理函数。
- **HandleSignRequest**: 处理签名请求。
- **HandleSignResponse**: 处理签名响应。
- **HandleSignResult**: 处理签名结果。

## 模块使用说明

1. **获取身份令牌**: 通过 Web 服务的登录接口获取 JWT 令牌。
2. **启动服务**: 使用 `NewServer` 创建服务器实例并调用 `Start` 方法启动服务。
3. **客户端连接**: 客户端通过 WebSocket 连接到服务器，发送 `register` 消息进行注册，必须包含有效的 JWT 令牌。
4. **密钥生成请求**: 
   - 协调方首先通过 Web 服务请求创建密钥生成任务并获取 key_id。
   - 使用获得的 key_id 通过 WebSocket 发送 `keygen_request` 消息。
   - 服务器处理并与参与方交互完成密钥生成。
5. **签名操作**: 
   - 协调方首先通过 Web 服务请求创建签名任务。
   - 通过 WebSocket 发送 `sign_request` 消息。
   - 服务器处理并与参与方交互完成签名。

## 安全注意事项

1. **身份验证**: 确保所有消息都包含有效的JWT令牌，并在服务端严格验证。
2. **传输安全**: 建议在生产环境中使用WSS(WebSocket Secure)，确保通信加密。
3. **访问控制**: 严格区分协调方和参与方的权限，避免越权操作。
4. **错误处理**: 对所有错误进行妥善处理，避免暴露敏感信息。
5. **日志安全**: 确保日志不包含敏感信息，如私钥或完整的JWT令牌。
