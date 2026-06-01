# Web 模块接口说明

Web 模块负责登录、用户管理、安全芯片登记，以及创建 keygen/sign 的 `session_key`。MPC 参数、分片、签名结果都通过 WebSocket 新协议传输，见 `ws_module_documentation.md`。

## 认证

登录：

```http
POST /user/login
Content-Type: application/json
```

```json
{
  "username": "admin",
  "password": "admin123"
}
```

返回 JWT 后，后续请求使用：

```http
Authorization: <jwt>
```

## 用户

注册：

```http
POST /user/register
```

```json
{
  "username": "u1",
  "password": "password",
  "email": "u1@example.com"
}
```

管理员查询用户：

```http
GET /user/admin/users
```

管理员更新角色：

```http
PUT /user/admin/users/:username/role
```

```json
{
  "role": "participant"
}
```

角色包括 `admin`、`coordinator`、`participant`、`auditor`。

## 安全芯片

登记安全芯片：

```http
POST /se/create
```

```json
{
  "se_id": "SE01",
  "cplc": "<security-element-cplc>",
  "custody_location": "柜台A"
}
```

`se_id` 是离线系统管理编号，`cplc` 是硬件返回的芯片标识。

## Keygen Session

创建 keygen 会话编号：

```http
GET /keygen/create/:initiator
```

返回：

```json
{
  "session_key": "keygen_20260528120000_admin"
}
```

随后协调端通过 WebSocket 发送 `keygen_request`，字段为 `required_signers`、`total_parties`、`participants`。

## Sign Session

创建 sign 会话编号：

```http
GET /sign/create/:initiator
```

返回：

```json
{
  "session_key": "sign_20260528121000_admin"
}
```

随后协调端通过 WebSocket 发送 `sign_request`，字段为 `message_hash`、`address`、`participants`。

## 离线任务包

导入在线任务包：

```http
POST /offline/tasks/import
Content-Type: application/json
```

也支持 `multipart/form-data` 的 `file` 字段。

查询任务：

```http
GET /offline/tasks/:task_no
```

返回的 `task` 使用 snake_case，例如：

```json
{
  "success": true,
  "task": {
    "task_no": "TASK-001",
    "task_type": "custody_keygen",
    "source_system": "online",
    "payload_hash": "sha256:...",
    "result_hash": "",
    "status": "imported"
  }
}
```

基于任务生成 WebSocket 消息模板：

```http
POST /offline/tasks/:task_no/keygen/start
POST /offline/tasks/:task_no/sign/start
```

下载结果包：

```http
GET /offline/results/:task_no/download
```

任务包必须满足：

- `schema_version = "1.0"`
- `package_type = "offline_task"`
- `task_type = "custody_keygen"` 或 `"sign"`
- `source_system = "online"`
- `target_system = "offline"`
- `payload_hash = sha256:<payload compact json hash>`

## 离线密钥

查询密钥：

```http
GET /offline/keys/:offline_key_id
```

返回的 `key` 使用 snake_case，例如：

```json
{
  "success": true,
  "key": {
    "offline_key_id": "OFFKEY-TASK-001",
    "task_no": "TASK-001",
    "address": "0x1111111111111111111111111111111111111111",
    "coin_type": "ETH",
    "algorithm": "GG20_ECDSA_SECP256K1",
    "required_signers": 2,
    "total_parties": 3,
    "logical_owner": "case-owner-a",
    "status": "active",
    "shards": [
      {
        "shard_id": "OFFKEY-TASK-001:1",
        "username": "u1",
        "shard_index": 1,
        "record_id": "64-byte-hex-string",
        "se_cplc": "<security-element-cplc>",
        "status": "active",
        "encrypted_blob_size": 4096,
        "encrypted_blob_sha256": "sha256:..."
      }
    ]
  }
}
```

查询接口只返回密文摘要和分布信息，不返回 local share 明文，也不返回完整 `encrypted_blob`。
每次查询都会写入 `audit_logs`，用于满足授权查询留痕要求。

销毁密钥：

```http
POST /offline/keys/:offline_key_id/destroy
```

请求体可传入 `reason` 和可选 `participants`。不传 `participants` 时，服务端会选择该密钥的全部 active shard：

```json
{
  "reason": "案件密钥销毁",
  "participants": ["u1", "u2", "u3"]
}
```

接口返回 `destroy_request` WebSocket 消息模板：

```json
{
  "success": true,
  "message": {
    "type": "destroy_request",
    "session_key": "destroy_20260528122000_OFFKEY-TASK-001",
    "offline_key_id": "OFFKEY-TASK-001",
    "address": "0x1111111111111111111111111111111111111111",
    "participants": ["u1", "u2", "u3"],
    "reason": "案件密钥销毁"
  }
}
```

协调端随后通过 WebSocket 发送该消息。服务端邀请参与方插入对应 SE，桌面端执行 DELETE 并验证 READ 失败后，服务端才把相关 `key_shards.status` 和 `offline_keys.status` 置为 `destroyed`。

## 审计和审批

审计查询允许 `admin`、`coordinator`、`auditor` 访问：

```http
GET /offline/audit?limit=100
GET /offline/approvals?limit=100
```

响应字段统一使用 snake_case：

```json
{
  "success": true,
  "logs": [
    {
      "created_at": "2026-05-28T12:00:00Z",
      "username": "admin",
      "role": "admin",
      "action": "offline_key_destroy",
      "resource_type": "offline_key",
      "resource_id": "OFFKEY-TASK-001",
      "result": "success",
      "sensitive_redacted": true
    }
  ]
}
```

移交、销毁等敏感操作会写入 `approvals` 和 `audit_logs`。当前原型采用“管理员发起即批准”的审批记录，以满足存管提控原型测试里的可追踪要求；正式生产可在同一数据模型上扩展为双人复核。
