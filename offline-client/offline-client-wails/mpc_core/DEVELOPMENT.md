# MPC Core 开发说明

`mpc_core` 是离线桌面端的本地执行层。它只接受离线服务端通过 WebSocket 下发的新协议参数，不再支持旧的 `user_name`、`encrypted_key`、`total_parts` 等调用方式。

## Keygen 调用

前端收到 `keygen_params` 后调用 `PerformKeyGeneration`：

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

本地执行：

```bash
gg20_keygen \
  --address <manager_addr> \
  --threshold <threshold> \
  --number-of-parties <parties> \
  --index <party_index> \
  --output <filename> \
  --room <room>
```

完成后：

1. 从 local share 提取公钥和地址。
2. 生成 32 字节 AES key。
3. 用 AES key 加密压缩后的 local share。
4. 将 AES key 写入 SE，索引为 `record_id + address`。
5. 返回 `address`、`public_key`、`record_id`、`encrypted_shard`。

## Signing 调用

前端收到 `sign_params` 后调用 `PerformSignMessage`：

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

本地执行：

```bash
gg20_signing \
  --address <manager_addr> \
  --index <signing_index> \
  --parties <parties> \
  --data-to-sign <message_hash> \
  --local-share <filename> \
  --room <room>
```

注意：

- `parties` 使用 keygen 时的原始 `party_index` 列表。
- `signing_index` 是当前参与方在 `parties` 中的 1-based 位置。
- 2-of-3 签名要支持 `1,2`、`1,3`、`2,3` 和 `1,2,3`。

## SE 访问

`SecurityService` 将 `record_id` 当作 32 字节 hex 解析，并直接传给 Applet 的第一段 32 字节字段。Applet 里历史字段名可能仍叫 `userName`，但上层语义统一为 `record_id`。

SE 数据模型：

```text
record_id(32 bytes) + address(20 bytes) -> aes_key(32 bytes)
```

读取和删除都必须携带服务端对 `record_id || address` 生成的授权签名。
