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

### 消息类型与交互流程

#### 注册流程

1. 客户端发送 `register` 消息：
   ```json
   {
       "type": "register",
       "payload": {
           "user_id": "client123",
           "role": "participant"
       }
   }
   ```
   - 参数说明：
     - `user_id`: 客户端的唯一标识符。
     - `role`: 客户端角色，支持 `coordinator` 或 `participant`。

2. 服务器返回确认消息：
   ```json
   {
       "type": "register_confirm",
       "payload": {
           "status": "success"
       }
   }
   ```

#### 密钥生成流程

1. 协调方发送 `keygen_request` 消息：
   ```json
   {
       "type": "keygen_request",
       "payload": {
           "key_id": "key123",
           "threshold": 2,
           "participants": ["client1", "client2"]
       }
   }
   ```
   - 参数说明：
     - `key_id`: 密钥的唯一标识符。
     - `threshold`: 密钥生成的阈值。
     - `participants`: 参与密钥生成的客户端列表。

2. 服务器向参与方发送 `keygen_invite` 消息：
   ```json
   {
       "type": "keygen_invite",
       "payload": {
           "key_id": "key123",
           "threshold": 2,
           "participants": ["client1", "client2"]
       }
   }
   ```

3. 参与方响应 `keygen_response` 消息：
   ```json
   {
       "type": "keygen_response",
       "payload": {
           "key_id": "key123",
           "response": true
       }
   }
   ```

4. 服务器发送 `keygen_params` 消息：
   ```json
   {
       "type": "keygen_params",
       "payload": {
           "key_id": "key123",
           "threshold": 2,
           "total_parts": 2,
           "part_index": 1,
           "output_file": "key123.json"
       }
   }
   ```

5. 参与方完成后发送 `keygen_complete` 消息：
   ```json
   {
       "type": "keygen_complete",
       "payload": {
           "key_id": "key123",
           "share_json": "{...}"
       }
   }
   ```

#### 签名流程

1. 协调方发送 `sign_request` 消息：
   ```json
   {
       "type": "sign_request",
       "payload": {
           "key_id": "key123",
           "data": "message_to_sign",
           "participants": ["client1", "client2"]
       }
   }
   ```

2. 服务器向参与方发送 `sign_invite` 消息：
   ```json
   {
       "type": "sign_invite",
       "payload": {
           "key_id": "key123",
           "data": "message_to_sign",
           "participants": ["client1", "client2"]
       }
   }
   ```

3. 参与方响应 `sign_response` 消息：
   ```json
   {
       "type": "sign_response",
       "payload": {
           "key_id": "key123",
           "response": true
       }
   }
   ```

4. 服务器发送 `sign_params` 消息：
   ```json
   {
       "type": "sign_params",
       "payload": {
           "key_id": "key123",
           "data": "message_to_sign",
           "participants": "1,2",
           "share_json": "{...}"
       }
   }
   ```

5. 参与方完成后发送 `sign_result` 消息：
   ```json
   {
       "type": "sign_result",
       "payload": {
           "key_id": "key123",
           "signature": "signed_message"
       }
   }
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
- **Message**: 基本消息结构，包含类型、用户 ID 和载荷。
- **RegisterPayload**: 注册消息的载荷。
- **KeyGenRequestPayload**: 密钥生成请求的载荷。
- **SignRequestPayload**: 签名请求的载荷。

## 模块使用说明

1. **启动服务**: 使用 `NewServer` 创建服务器实例并调用 `Start` 方法启动服务。
2. **客户端连接**: 客户端通过 WebSocket 连接到服务器，发送 `register` 消息进行注册。
3. **密钥生成**: 协调方发送 `keygen_request` 消息，服务器处理并与参与方交互完成密钥生成。
4. **签名操作**: 协调方发送 `sign_request` 消息，服务器处理并与参与方交互完成签名。
