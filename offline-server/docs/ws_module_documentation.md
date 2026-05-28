# WS 模块协议说明

本文件描述离线系统当前唯一支持的 WebSocket 协议。旧字段 `total_parts`、`part_index`、`cpic` 不再作为接口字段使用；新协议统一使用 `total_parties`、`party_index`、`cplc` 和 `record_id`。

## 注册

客户端连接 `/ws` 后先注册：

```json
{
  "type": "register",
  "username": "u1",
  "role": "participant",
  "token": "<jwt>"
}
```

服务端返回：

```json
{
  "type": "register_complete",
  "success": true,
  "message": "注册成功"
}
```

## Keygen

协调方发起：

```json
{
  "type": "keygen_request",
  "session_key": "keygen_20260528120000_admin",
  "task_no": "TASK-001",
  "offline_key_id": "OFFKEY-001",
  "required_signers": 2,
  "total_parties": 3,
  "participants": ["u1", "u2", "u3"]
}
```

服务端行为：

1. 为本次会话启动独立 `gg20_sm_manager --address <bind> --port <port>`。
2. 生成本次 `manager_addr` 和 `room`。
3. 为每个参与方分配安全芯片和 `party_index`。

参与方收到邀请：

```json
{
  "type": "keygen_invite",
  "session_key": "keygen_20260528120000_admin",
  "coordinator": "admin",
  "required_signers": 2,
  "total_parties": 3,
  "party_index": 1,
  "se_id": "SE01",
  "participants": ["u1", "u2", "u3"]
}
```

参与方接受：

```json
{
  "type": "keygen_response",
  "session_key": "keygen_20260528120000_admin",
  "party_index": 1,
  "cplc": "<security-element-cplc>",
  "accept": true
}
```

全部接受后，服务端下发执行参数：

```json
{
  "type": "keygen_params",
  "session_key": "keygen_20260528120000_admin",
  "manager_addr": "http://192.168.1.10:18001",
  "room": "keygen_20260528120000_admin",
  "threshold": 1,
  "total_parties": 3,
  "party_index": 1,
  "record_id": "64-byte-hex-string",
  "filename": "keygen_20260528120000_admin_keygen_1.json"
}
```

桌面端执行 `gg20_keygen`，将 AES key 写入 SE，并返回加密后的 local share：

```json
{
  "type": "keygen_result",
  "session_key": "keygen_20260528120000_admin",
  "party_index": 1,
  "address": "0x1111111111111111111111111111111111111111",
  "public_key": "<public-key>",
  "cplc": "<security-element-cplc>",
  "record_id": "64-byte-hex-string",
  "encrypted_shard": "<base64>",
  "success": true,
  "message": "ok"
}
```

所有分片完成后，服务端保存 `offline_keys` 和 `key_shards`，停止本次 manager，并通知协调方：

```json
{
  "type": "keygen_complete",
  "session_key": "keygen_20260528120000_admin",
  "address": "0x1111111111111111111111111111111111111111",
  "success": true,
  "message": "密钥生成已完成"
}
```

## Signing

协调方发起：

```json
{
  "type": "sign_request",
  "session_key": "sign_20260528121000_admin",
  "task_no": "TASK-002",
  "offline_key_id": "OFFKEY-001",
  "transaction_no": "TX-001",
  "message_hash": "0000000000000000000000000000000000000000000000000000000000000001",
  "address": "0x1111111111111111111111111111111111111111",
  "participants": ["u2", "u3"]
}
```

服务端行为：

1. 校验离线密钥状态和参与人数。
2. 按参与者读取原始 `shard_index`。
3. 对 `shard_index` 排序生成 `parties`，例如参与者 `u2 + u3` 得到 `"2,3"`。
4. 为本次签名启动独立 manager。

参与方收到邀请：

```json
{
  "type": "sign_invite",
  "session_key": "sign_20260528121000_admin",
  "message_hash": "0000000000000000000000000000000000000000000000000000000000000001",
  "address": "0x1111111111111111111111111111111111111111",
  "party_index": 2,
  "se_id": "SE02",
  "participants": ["u2", "u3"]
}
```

参与方接受：

```json
{
  "type": "sign_response",
  "session_key": "sign_20260528121000_admin",
  "party_index": 2,
  "cplc": "<security-element-cplc>",
  "accept": true
}
```

全部接受后，服务端下发签名参数：

```json
{
  "type": "sign_params",
  "session_key": "sign_20260528121000_admin",
  "manager_addr": "http://192.168.1.10:18002",
  "room": "sign_20260528121000_admin",
  "message_hash": "0000000000000000000000000000000000000000000000000000000000000001",
  "address": "0x1111111111111111111111111111111111111111",
  "signature": "<base64-se-authorization-signature>",
  "parties": "2,3",
  "party_index": 2,
  "signing_index": 1,
  "record_id": "64-byte-hex-string",
  "filename": "sign_20260528121000_admin_sign_1.json",
  "encrypted_shard": "<base64>"
}
```

`party_index` 是 keygen 时的原始分片编号；`signing_index` 是当前分片在 `parties` 中的 1-based 位置。2-of-3 签名必须支持 `1,2`、`1,3`、`2,3`，也允许 `1,2,3` 一起参与。

桌面端返回：

```json
{
  "type": "sign_result",
  "session_key": "sign_20260528121000_admin",
  "signing_index": 1,
  "success": true,
  "signature": "0x...",
  "message": "ok"
}
```

所有参与方完成后，服务端停止本次 manager，并通知协调方：

```json
{
  "type": "sign_complete",
  "session_key": "sign_20260528121000_admin",
  "signature": "0x...",
  "success": true,
  "message": "签名已完成"
}
```

多个参与方返回的 `signature` 必须一致。若任一成功结果与已收到的签名不同，服务端会将会话标记为 failed，通知协调方并停止本次 manager。

## Destroy

管理员从 HTTP 接口拿到 `destroy_request` 后，通过 WebSocket 发送：

```json
{
  "type": "destroy_request",
  "session_key": "destroy_20260528122000_OFFKEY-001",
  "offline_key_id": "OFFKEY-001",
  "address": "0x1111111111111111111111111111111111111111",
  "participants": ["u1", "u2", "u3"],
  "reason": "案件密钥销毁"
}
```

服务端按 active `key_shards` 邀请相关参与方：

```json
{
  "type": "destroy_invite",
  "session_key": "destroy_20260528122000_OFFKEY-001",
  "offline_key_id": "OFFKEY-001",
  "address": "0x1111111111111111111111111111111111111111",
  "party_index": 1,
  "se_id": "SE01",
  "reason": "案件密钥销毁"
}
```

参与方插入对应 SE，确认后返回 CPLC：

```json
{
  "type": "destroy_response",
  "session_key": "destroy_20260528122000_OFFKEY-001",
  "party_index": 1,
  "cplc": "<security-element-cplc>",
  "accept": true
}
```

所有参与方确认后，服务端下发删除授权：

```json
{
  "type": "destroy_params",
  "session_key": "destroy_20260528122000_OFFKEY-001",
  "offline_key_id": "OFFKEY-001",
  "address": "0x1111111111111111111111111111111111111111",
  "party_index": 1,
  "record_id": "64-byte-hex-string",
  "signature": "<base64-se-delete-authorization-signature>"
}
```

桌面端执行 SE DELETE，并立即 READ 验证不可读取，成功后返回：

```json
{
  "type": "destroy_result",
  "session_key": "destroy_20260528122000_OFFKEY-001",
  "party_index": 1,
  "success": true,
  "message": "SE记录已删除并验证不可读取"
}
```

服务端收到所有成功结果后，才把对应 `key_shards.status` 和 `offline_keys.status` 标记为 `destroyed`，并通知发起方：

```json
{
  "type": "destroy_complete",
  "session_key": "destroy_20260528122000_OFFKEY-001",
  "offline_key_id": "OFFKEY-001",
  "address": "0x1111111111111111111111111111111111111111",
  "destroyed": 3,
  "success": true,
  "message": "密钥销毁已完成"
}
```

## 错误

任何阶段失败都返回：

```json
{
  "type": "error",
  "message": "错误摘要",
  "details": "错误详情"
}
```

失败、拒绝、会话完成或服务退出时，服务端都必须清理本次 manager。
