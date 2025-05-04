# WS模块开发文档

## 简介

WS模块是一个基于WebSocket的多方门限签名系统，提供了安全的分布式密钥生成和签名功能。该模块支持多角色协作，通过安全芯片保护私钥分片，实现了高安全性的加密操作。

## 系统角色

系统支持两种角色：

1. **协调者(Coordinator)**: 负责发起密钥生成和签名请求
2. **参与者(Participant)**: 负责响应协调者的请求，参与密钥生成和签名过程

## 主要功能

1. **分布式密钥生成**：生成多方持有的私钥分片，支持门限签名
2. **分布式签名**：使用多方持有的私钥分片进行签名

## 连接与认证

### 建立WebSocket连接

1. 连接到WebSocket服务器
2. 发送注册消息进行身份验证

```json
{
  "type": "register",
  "username": "您的用户名",
  "role": "coordinator或participant",
  "token": "您的JWT令牌"
}
```

服务器将返回注册完成消息：

```json
{
  "type": "register_complete",
  "success": true,
  "message": "注册成功"
}
```

## 密钥生成流程

### 1. 协调者发起密钥生成请求

协调者需要发送以下消息：

```json
{
  "type": "keygen_request",
  "session_key": "唯一会话标识",
  "threshold": 3,
  "total_parts": 5,
  "participants": ["用户1", "用户2", "用户3", "用户4", "用户5"]
}
```

参数说明：
- `session_key`: 唯一标识此次密钥生成会话
- `threshold`: 门限值，需要至少多少个分片才能重构私钥（进行签名）
- `total_parts`: 总分片数，将私钥分成多少份
- `participants`: 参与密钥生成的用户名列表

### 2. 参与者接收密钥生成邀请

参与者将收到以下邀请消息：

```json
{
  "type": "keygen_invite",
  "session_key": "会话标识",
  "coordinator": "协调者用户名",
  "threshold": 3,
  "total_parts": 5,
  "part_index": 2,
  "se_id": "安全芯片标识符",
  "participants": ["用户1", "用户2", "用户3", "用户4", "用户5"]
}
```

### 3. 参与者响应密钥生成邀请

参与者需要回复是否接受参与：

```json
{
  "type": "keygen_response",
  "session_key": "会话标识",
  "part_index": 2,
  "cpic": "安全芯片唯一标识符",
  "accept": true,
  "reason": "" // 如果拒绝，需提供原因
}
```

### 4. 参与者接收密钥生成参数

当所有参与者都接受邀请后，参与者将收到生成参数：

```json
{
  "type": "keygen_params",
  "session_key": "会话标识",
  "threshold": 3,
  "total_parts": 5,
  "part_index": 2,
  "filename": "密钥生成配置文件名"
}
```

### 5. 参与者发送密钥生成结果

参与者完成密钥生成后，需发送结果：

```json
{
  "type": "keygen_result",
  "session_key": "会话标识",
  "part_index": 2,
  "address": "生成的账户地址",
  "cpic": "安全芯片唯一标识符",
  "encrypted_shard": "Base64编码的加密密钥分片",
  "success": true,
  "message": "密钥生成成功"
}
```

### 6. 协调者接收密钥生成完成通知

当所有参与者都完成密钥生成后，协调者将收到完成通知：

```json
{
  "type": "keygen_complete",
  "session_key": "会话标识",
  "address": "生成的账户地址",
  "success": true,
  "message": "密钥生成成功"
}
```

## 签名流程

### 1. 协调者发起签名请求

协调者需要发送以下消息：

```json
{
  "type": "sign_request",
  "session_key": "唯一会话标识",
  "threshold": 3,
  "total_parts": 5,
  "data": "要签名的数据(32字节的哈希值)",
  "address": "账户地址",
  "participants": ["用户1", "用户2", "用户3"]
}
```

参数说明：
- `session_key`: 唯一标识此次签名会话
- `threshold`: 门限值
- `total_parts`: 总分片数
- `data`: 要签名的数据(32字节的哈希值)
- `address`: 账户地址（表示使用哪个账户的密钥）
- `participants`: 选定参与签名的用户列表（数量必须≥门限值）

### 2. 参与者接收签名邀请

参与者将收到以下邀请消息：

```json
{
  "type": "sign_invite",
  "session_key": "会话标识",
  "data": "要签名的数据",
  "address": "账户地址",
  "part_index": 2,
  "se_id": "安全芯片标识符",
  "participants": ["用户1", "用户2", "用户3"]
}
```

### 3. 参与者响应签名邀请

参与者需要回复是否接受参与：

```json
{
  "type": "sign_response",
  "session_key": "会话标识",
  "part_index": 2,
  "cpic": "安全芯片唯一标识符",
  "accept": true,
  "reason": "" // 如果拒绝，需提供原因
}
```

### 4. 参与者接收签名参数

当所有参与者都接受邀请后，参与者将收到签名参数：

```json
{
  "type": "sign_params",
  "session_key": "会话标识",
  "data": "要签名的数据(Base64编码)",
  "address": "账户地址",
  "signature": "用于从安全芯片中获取私钥分片的签名",
  "parties": "参与者列表(逗号分隔的索引)",
  "part_index": 2,
  "filename": "签名配置文件名",
  "encrypted_shard": "Base64编码的加密密钥分片"
}
```

### 5. 参与者发送签名结果

参与者完成签名后，需发送结果：

```json
{
  "type": "sign_result",
  "session_key": "会话标识",
  "part_index": 2,
  "success": true,
  "signature": "签名结果",
  "message": "签名成功"
}
```

### 6. 协调者接收签名完成通知

当所有参与者都完成签名后，协调者将收到完成通知：

```json
{
  "type": "sign_complete",
  "session_key": "会话标识",
  "signature": "最终签名结果",
  "success": true,
  "message": "签名成功"
}
```

## 错误处理

在操作过程中，可能会收到错误消息：

```json
{
  "type": "error",
  "message": "错误消息",
  "details": "错误详情"
}
```

## 注意事项

1. **会话标识唯一性**: 每次密钥生成或签名操作，`session_key` 必须唯一
2. **安全芯片管理**: 确保安全芯片标识符 (`se_id`) 安全保存
3. **门限值设置**: 门限值 (`threshold`) 应根据安全需求设置，建议不低于总分片数的一半
4. **错误处理**: 实现完善的错误处理逻辑，尤其是网络断开重连机制
5. **参与者数量**: 签名时的参与者数量必须大于或等于门限值

## 安全建议

1. 使用安全通信通道，如TLS加密的WebSocket连接 (WSS)
2. 定期更新JWT令牌
3. 实施访问控制政策，限制用户权限
4. 记录审计日志，跟踪所有密钥生成和签名操作
5. 实施防DDoS措施
6. 定期安全审查系统

## 常见问题与解决方案

1. **问题**: 连接断开后如何恢复会话？
   **解决方案**: 实现重连机制，保存会话状态，重连后发送新的注册消息

2. **问题**: 如何处理参与者拒绝参与的情况？
   **解决方案**: 协调者需要重新选择参与者列表，发起新的请求

3. **问题**: 密钥生成或签名过程中出错怎么办？
   **解决方案**: 实现错误恢复机制，出错时通知所有参与者中止当前会话

4. **问题**: 如何验证生成的密钥对应的公钥是否正确？
   **解决方案**: 使用生成的地址与已知的公钥地址进行比对

5. **问题**: 性能优化建议？
   **解决方案**: 实现消息批处理，优化网络传输，使用高效的加密库 