# ws 目录

`ws` 目录实现离线协作服务端的 WebSocket 协调服务，用于连接桌面客户端并协调 MPC 密钥生成和签名消息。

## 文件说明

- `server.go`：WebSocket 服务启动、连接升级和关闭。
- `hub.go`：连接管理、广播和客户端状态维护。
- `client.go`：单个客户端连接的读写循环。
- `handler.go`：消息分发入口。
- `keygen_handler.go`：密钥生成消息处理。
- `sign_handler.go`：签名消息处理。
- `crypto.go`：协作流程中使用的加密辅助逻辑。
- `types.go`：消息结构和类型定义。
- `storage/`：WebSocket 流程使用的轻量存储辅助。

## 连接地址

```text
ws://localhost:8081/ws
```

## 维护建议

- 新增消息类型时同步更新 `types.go`、消息处理器和 `docs/ws_module_documentation.md`。
- 连接断开、重复连接和会话超时需要保持可恢复。
- 日志中不要记录私钥分片、签名中间值和完整认证令牌。
