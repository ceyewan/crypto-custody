# storage 目录

该目录保存 WebSocket 协调流程使用的轻量存储辅助逻辑，用于记录连接、会话或消息处理所需的临时状态。

## 文件说明

- `storage.go`：WebSocket 消息处理过程中的存储封装。

## 维护建议

- 临时状态应与持久化数据边界清晰，长期数据放入顶层 `storage` 层。
- 修改会话状态字段后，同步检查 `keygen_handler.go` 和 `sign_handler.go`。
