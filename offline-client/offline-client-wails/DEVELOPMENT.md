# Offline Client Wails 开发说明

桌面端负责三件事：

1. 连接离线服务端 WebSocket，接收 keygen/sign 邀请和执行参数。
2. 调用内置 `gg20_keygen`、`gg20_signing` 二进制完成 MPC 运算。
3. 通过 SE 保存、读取或删除本地 AES key。

协议以离线服务端下发的 `keygen_params` 和 `sign_params` 为准，不再支持旧字段。

## Keygen 参数

```json
{
  "manager_addr": "http://192.168.1.10:18001",
  "room": "keygen-session",
  "threshold": 1,
  "parties": 3,
  "party_index": 1,
  "record_id": "64-byte-hex-string",
  "filename": "keygen-session_keygen_1.json"
}
```

`threshold` 是 GG20 参数，即业务门限人数减 1。

## Sign 参数

```json
{
  "manager_addr": "http://192.168.1.10:18002",
  "room": "sign-session",
  "parties": "1,3",
  "signing_index": 2,
  "message_hash": "32-byte-hex-message-hash",
  "filename": "sign-session_sign_2.json",
  "encrypted_shard": "<base64>",
  "record_id": "64-byte-hex-string",
  "address": "0x1111111111111111111111111111111111111111",
  "signature": "<base64-se-authorization-signature>"
}
```

`party_index` 是 keygen 原始分片编号；`signing_index` 是当前参与方在 `parties` 中的位置。

## 本地执行链路

```text
frontend ws.js
  -> wails-api.js
  -> App
  -> WailsServices
  -> MPCService
  -> utils.RunKeyGen / utils.RunSigning
  -> SecurityService / seclient
```

## 前端页面

- `OfflineTasks.vue`：导入在线任务包、生成并发送 keygen/sign WebSocket 请求、下载离线结果包。
- `KeyManagement.vue`：查询离线密钥、执行管理员移交，发起销毁 WebSocket 流程。
- `AuditLogs.vue`：查询审计日志和敏感操作审批记录，允许 admin/coordinator/auditor 访问。

离线 HTTP API 返回字段统一使用 snake_case，例如 `task_no`、`offline_key_id`、`created_at`。

## Destroy 参数

服务端下发 `destroy_params` 后，桌面端调用 `PerformDeleteMessage`：

```json
{
  "record_id": "64-byte-hex-string",
  "address": "0x1111111111111111111111111111111111111111",
  "signature": "<base64-se-delete-authorization-signature>"
}
```

删除成功后桌面端会立即尝试 READ，同一记录不可读取时才回传 `destroy_result.success = true`。

## 测试

```bash
go test ./mpc_core/...
cd frontend && npm run build
```

客户端单元测试覆盖：

- `gg20_keygen` 参数包含 `manager_addr`、`room`、`threshold`、`party_index`。
- `gg20_signing` 使用 `signing_index` 和原始 `parties`。
- `record_id` 必须是 32 字节 hex，`address` 必须是 20 字节 hex。
