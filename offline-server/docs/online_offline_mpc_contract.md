# 在线离线数据包与 MPC 边界设计

## 1. 文档目的

本文档定义在线系统、离线服务端、离线桌面端和安全芯片之间的数据边界、交换包格式、数据库职责以及 GG20 keygen/sign 流程。

本文档是目标契约，用于指导后续代码改造。当前实现中如果存在与本文档不一致的地方，应以本文档为准逐步修正。

## 2. 总体原则

- 在线系统是业务主数据源，负责案件、账户、交易、审批、链上查询、交易构建、签名结果导入和广播。
- 离线系统是密钥安全域，负责门限密钥生成、加密分片保存、安全芯片授权和门限签名。
- 在线系统和离线系统不做实时同步，只通过 JSON 任务包和 JSON 结果包交换必要数据。
- 在线系统不得接触私钥、助记词、明文 share、加密 share、SE 读取授权签名、参与方和安全芯片详细映射。
- 离线系统不得负责链上 nonce、gas、余额查询和交易广播。
- 离线系统内部可以保存加密后的 GG20 share，但解密材料必须由参与方安全芯片控制。

一句话边界：

```text
在线导出可验证任务，离线执行密钥操作，离线导出可验证结果；MPC 分片和 SE 授权留在离线域内部。
```

## 3. 数据分类

### 3.1 在线离线交换数据

这类数据可以通过 U 盘、二维码、文件上传下载等方式在在线系统和离线系统之间传递。

| 数据 | 方向 | 说明 |
| --- | --- | --- |
| 托管钱包生成任务包 | 在线 -> 离线 | 案件编号、币种、门限策略、任务编号 |
| 托管钱包生成结果包 | 离线 -> 在线 | 托管钱包地址、公钥、任务编号、结果摘要 |
| 签名任务包 | 在线 -> 离线 | 交易编号、未签名交易、待签名哈希、业务展示字段 |
| 签名结果包 | 离线 -> 在线 | 签名值、消息哈希、任务编号、结果摘要 |

### 3.2 离线域内部数据

这类数据不得进入在线系统。

| 数据 | 保存位置 | 说明 |
| --- | --- | --- |
| `local-share.json` 明文 | 桌面端临时文件 | 只在 keygen/sign 过程中短暂存在，结束后删除 |
| `encrypted_shard` | 离线服务端数据库 | AES-GCM 加密后的 GG20 share |
| AES-GCM 解密密钥 | 参与者 SE | 用于解密 `encrypted_shard`，不进入服务端数据库 |
| SE 读取授权签名 | 离线服务端临时生成并下发 | 用于授权本次读取 SE 中的解密密钥 |
| CPLC / SE ID 映射 | 离线服务端数据库 | 属于离线审计和芯片管理数据 |
| gg20 `manager_addr` / `room` / `parties` / `signing_index` | 离线服务端与桌面端 | 只用于本次 MPC 会话 |
| 参与方名单和 shard index | 离线服务端数据库 | 在线系统只保存离线引用号和结果摘要 |

## 4. JSON 包格式约定

### 4.1 基础格式

- 文件编码：UTF-8。
- 文件类型：`.json`。
- 字段命名：`snake_case`。
- 时间格式：RFC3339 UTC，例如 `2026-05-28T12:00:00Z`。
- 金额：十进制字符串，不使用浮点数。
- 地址：以 `0x` 开头的十六进制字符串。在线系统导入时可做 checksum 校验。
- 哈希：以 `0x` 开头的 32 字节十六进制字符串。
- 以太坊签名：`0x` + 65 字节 RSV 十六进制字符串。
- 包哈希：对 `payload` 的规范 JSON 字节计算 SHA-256，格式为 `sha256:<hex>`。

### 4.2 通用包信封

所有任务包和结果包都使用同一个外层结构。

```json
{
  "schema_version": "1.0",
  "package_type": "offline_task",
  "task_type": "sign",
  "task_no": "TASK-2026-0002",
  "source_system": "online",
  "target_system": "offline",
  "created_by": "online-operator",
  "created_at": "2026-05-28T12:10:00Z",
  "payload": {},
  "payload_hash": "sha256:...",
  "package_signature": {
    "algorithm": "ecdsa-p256-sha256",
    "key_id": "online-system-key-1",
    "signature": "base64..."
  }
}
```

字段说明：

| 字段 | 说明 |
| --- | --- |
| `schema_version` | 包格式版本，当前为 `1.0` |
| `package_type` | `offline_task` 或 `offline_result` |
| `task_type` | `custody_keygen`、`custody_keygen_result`、`sign`、`sign_result` |
| `task_no` | 在线系统生成的任务编号，在线离线全流程唯一 |
| `source_system` | `online` 或 `offline` |
| `target_system` | `offline` 或 `online` |
| `created_by` | 创建包的用户或系统账号 |
| `created_at` | 创建时间 |
| `payload` | 具体业务数据 |
| `payload_hash` | `payload` 的 SHA-256 摘要 |
| `package_signature` | 包签名，用于防篡改和来源确认 |

原型阶段如果暂未实现系统级包签名，仍应保留字段，允许为空对象，但必须先实现 `payload_hash` 校验。

## 5. 在线系统上传下载能力

在线系统需要提供以下用户操作或 API：

| 操作 | 方向 | 结果 |
| --- | --- | --- |
| 下载托管钱包生成任务包 | 在线 -> 文件 | 生成 `offline_task_<task_no>.json` |
| 上传托管钱包生成结果包 | 文件 -> 在线 | 校验后保存 custody address |
| 下载签名任务包 | 在线 -> 文件 | 生成 `offline_task_<task_no>.json` |
| 上传签名结果包 | 文件 -> 在线 | 校验后保存签名并广播交易 |

建议在线系统 API：

| API | 方法 | 说明 |
| --- | --- | --- |
| `/api/offline-tasks/keygen` | `POST` | 创建托管钱包生成离线任务 |
| `/api/offline-tasks/sign` | `POST` | 创建签名离线任务 |
| `/api/offline-tasks/{task_no}/download` | `GET` | 下载任务包 JSON 文件 |
| `/api/offline-results/upload` | `POST multipart/form-data` | 上传离线结果包 JSON 文件 |
| `/api/offline-tasks/{task_no}` | `GET` | 查询离线任务状态和摘要 |

建议文件名：

```text
offline_task_<task_no>.json
offline_result_<task_no>.json
```

在线系统导入结果包时必须校验：

- `schema_version` 是否支持。
- `package_type` 和 `task_type` 是否匹配当前操作。
- `task_no` 是否存在且状态允许导入。
- `payload_hash` 是否匹配。
- `package_signature` 是否来自可信离线系统。
- keygen 结果中的 `case_no` 是否和原任务一致。
- sign 结果中的 `transaction_no`、`from_address`、`message_hash` 是否和原任务一致。
- sign 结果中的签名能否恢复出 `from_address`。

在线系统不得接受或保存：

- `encrypted_shard`。
- `local_share`。
- SE 授权签名。
- CPLC 明细。
- 参与方名单。
- gg20 内部参数。

## 6. 离线系统上传下载能力

离线系统需要提供以下用户操作或 API：

| 操作 | 方向 | 结果 |
| --- | --- | --- |
| 上传或导入托管钱包生成任务包 | 文件 -> 离线 | 创建 OfflineTask，待协调员发起 keygen |
| 下载或导出托管钱包生成结果包 | 离线 -> 文件 | 生成 `offline_result_<task_no>.json` |
| 上传或导入签名任务包 | 文件 -> 离线 | 创建 OfflineTask，待协调员发起 sign |
| 下载或导出签名结果包 | 离线 -> 文件 | 生成 `offline_result_<task_no>.json` |

建议离线系统 API：

| API | 方法 | 说明 |
| --- | --- | --- |
| `/api/offline/tasks/import` | `POST multipart/form-data` | 导入在线任务包 JSON 文件 |
| `/api/offline/tasks/{task_no}` | `GET` | 查询导入任务、会话和结果摘要 |
| `/api/offline/tasks/{task_no}/keygen/start` | `POST` | 协调员基于任务发起 keygen 会话 |
| `/api/offline/tasks/{task_no}/sign/start` | `POST` | 协调员基于任务发起 sign 会话 |
| `/api/offline/results/{task_no}/download` | `GET` | 下载离线结果包 JSON 文件 |

离线系统导入任务包时必须校验：

- `package_type = offline_task`。
- `source_system = online`。
- `target_system = offline`。
- `payload_hash` 正确。
- 在线系统包签名可信。
- `task_no` 未重复导入，或重复导入内容完全一致。

离线系统导出结果包时必须写入：

- 原 `task_no`。
- 原 `case_no` 或 `transaction_no`。
- 结果摘要。
- 离线引用号 `offline_ref_no`。
- `payload_hash`。
- 离线系统包签名。

## 7. 托管钱包生成任务包

在线系统导出，离线系统导入。

```json
{
  "schema_version": "1.0",
  "package_type": "offline_task",
  "task_type": "custody_keygen",
  "task_no": "TASK-2026-0001",
  "source_system": "online",
  "target_system": "offline",
  "created_by": "online-operator",
  "created_at": "2026-05-28T12:00:00Z",
  "payload": {
    "case_no": "CASE-2026-001",
    "coin_type": "ETH",
    "chain_id": "1",
    "threshold_policy": {
      "required_signers": 2,
      "total_parties": 3
    },
    "business_reason": "创建案件托管钱包"
  },
  "payload_hash": "sha256:...",
  "package_signature": {
    "algorithm": "ecdsa-p256-sha256",
    "key_id": "online-system-key-1",
    "signature": "base64..."
  }
}
```

说明：

- `required_signers` 是业务口径，例如 2-of-3 中的 `2`。
- 离线服务端调用 GG20 时必须转换为 `gg20_threshold = required_signers - 1`。
- 在线系统不指定参与警员和安全芯片，由离线协调员在离线域内选择。

## 8. 托管钱包生成结果包

离线系统导出，在线系统导入。

```json
{
  "schema_version": "1.0",
  "package_type": "offline_result",
  "task_type": "custody_keygen_result",
  "task_no": "TASK-2026-0001",
  "source_system": "offline",
  "target_system": "online",
  "created_by": "offline-coordinator",
  "created_at": "2026-05-28T12:20:00Z",
  "payload": {
    "case_no": "CASE-2026-001",
    "coin_type": "ETH",
    "chain_id": "1",
    "custody_address": "0x...",
    "public_key": "0x...",
    "threshold_policy": {
      "required_signers": 2,
      "total_parties": 3
    },
    "offline_ref_no": "OFFLINE-KEY-2026-0001",
    "completed_at": "2026-05-28T12:18:00Z"
  },
  "payload_hash": "sha256:...",
  "package_signature": {
    "algorithm": "ecdsa-p256-sha256",
    "key_id": "offline-system-key-1",
    "signature": "base64..."
  }
}
```

结果包不得包含：

- 参与方用户名。
- 安全芯片编号或 CPLC。
- `encrypted_shard`。
- GG20 share 明文或密文。

在线系统导入后只保存 `custody_address`、`public_key`、`offline_ref_no`、`payload_hash` 和任务状态。

## 9. 签名任务包

在线系统导出，离线系统导入。

```json
{
  "schema_version": "1.0",
  "package_type": "offline_task",
  "task_type": "sign",
  "task_no": "TASK-2026-0002",
  "source_system": "online",
  "target_system": "offline",
  "created_by": "online-operator",
  "created_at": "2026-05-28T12:30:00Z",
  "payload": {
    "case_no": "CASE-2026-001",
    "transaction_no": "TX-2026-0001",
    "coin_type": "ETH",
    "chain_id": "1",
    "from_address": "0x...",
    "to_address": "0x...",
    "value": "0.01",
    "unsigned_payload": "0x...",
    "message_hash": "0x...",
    "reason": "案件资产提取",
    "display": {
      "asset": "ETH",
      "amount": "0.01",
      "fee_limit": "0.001",
      "recipient_label": "案件资金接收账户"
    }
  },
  "payload_hash": "sha256:...",
  "package_signature": {
    "algorithm": "ecdsa-p256-sha256",
    "key_id": "online-system-key-1",
    "signature": "base64..."
  }
}
```

说明：

- 离线系统只签 `message_hash`。
- `unsigned_payload` 用于离线复核和在线导入后拼装广播交易。
- `display` 用于离线桌面端展示给参与者确认，不能只展示 hash。
- 离线系统不查询 nonce、gas、余额，不修改交易内容。

## 10. 签名结果包

离线系统导出，在线系统导入。

```json
{
  "schema_version": "1.0",
  "package_type": "offline_result",
  "task_type": "sign_result",
  "task_no": "TASK-2026-0002",
  "source_system": "offline",
  "target_system": "online",
  "created_by": "offline-coordinator",
  "created_at": "2026-05-28T12:45:00Z",
  "payload": {
    "case_no": "CASE-2026-001",
    "transaction_no": "TX-2026-0001",
    "coin_type": "ETH",
    "chain_id": "1",
    "from_address": "0x...",
    "message_hash": "0x...",
    "signature": "0x...",
    "signature_format": "ethereum_rsv",
    "offline_ref_no": "OFFLINE-SIGN-2026-0001",
    "completed_at": "2026-05-28T12:43:00Z"
  },
  "payload_hash": "sha256:...",
  "package_signature": {
    "algorithm": "ecdsa-p256-sha256",
    "key_id": "offline-system-key-1",
    "signature": "base64..."
  }
}
```

在线系统导入后必须：

- 用 `message_hash` 和 `signature` 恢复地址。
- 确认恢复地址等于 `from_address`。
- 确认 `task_no`、`transaction_no` 和原签名任务一致。
- 将签名写入交易记录。
- 广播交易并更新链上状态。

## 11. 离线服务端设计

### 11.1 职责

离线服务端负责：

- 离线用户认证和权限。
- 任务包导入、结果包导出。
- 安全芯片登记和状态管理。
- Keygen/sign 会话创建、邀请、参数下发、结果汇总。
- `gg20_sm_manager` 会话级生命周期管理。
- 加密 share 存储。
- 离线审计日志。

离线服务端不负责：

- 运行 `gg20_keygen` / `gg20_signing`。
- 保存明文 share。
- 保存 SE 内 AES key。
- 广播交易。
- 维护在线案件主数据。

### 11.2 组件

```text
offline-server
  |
  +-- Web API
  |     +-- 用户登录
  |     +-- 任务包导入
  |     +-- 结果包导出
  |     +-- 安全芯片登记
  |
  +-- WebSocket Hub
  |     +-- 桌面端注册
  |     +-- 邀请参与方
  |     +-- 下发 keygen/sign 参数
  |     +-- 收集 keygen/sign 结果
  |
  +-- GG20 Manager Runtime
  |     +-- 每个 keygen/sign 会话启动独立 manager
  |     +-- 分配端口和 manager_addr
  |     +-- 会话结束或失败后停止进程
  |
  +-- Storage
        +-- SQLite
        +-- 审计日志
```

### 11.3 数据库

建议 SQLite 表或模型如下。

#### users

| 字段 | 说明 |
| --- | --- |
| `username` | 离线用户名 |
| `password_hash` | 密码哈希 |
| `role` | `admin` / `coordinator` / `participant` / `auditor` |
| `status` | `active` / `disabled` |

#### security_chips

| 字段 | 说明 |
| --- | --- |
| `se_id` | 业务编号 |
| `cplc` | 芯片唯一标识 |
| `owner` | 持有人 |
| `status` | `active` / `lost` / `disabled` / `destroyed` |
| `registered_by` | 登记人 |

#### key_shards

| 字段 | 说明 |
| --- | --- |
| `address` | 托管钱包地址 |
| `username` | 分片持有人 |
| `shard_index` | keygen 时的原始 party index |
| `cplc` | 对应安全芯片 |
| `encrypted_shard` | AES-GCM 加密后的 GG20 local share |
| `status` | `active` / `transferred` / `destroyed` |

#### offline_tasks

| 字段 | 说明 |
| --- | --- |
| `task_no` | 在线任务编号 |
| `task_type` | `custody_keygen` / `sign` |
| `payload_hash` | 导入任务包 payload 哈希 |
| `result_hash` | 导出结果包 payload 哈希 |
| `status` | `imported` / `processing` / `completed` / `failed` |
| `raw_package_path` | 可选，原始任务包存档路径 |

#### keygen_sessions

| 字段 | 说明 |
| --- | --- |
| `session_key` | 离线 keygen 会话编号 |
| `task_no` | 关联任务编号 |
| `required_signers` | 业务门限人数 |
| `total_parties` | 总参与方数量 |
| `gg20_threshold` | `required_signers - 1` |
| `manager_addr` | 本会话 manager 地址 |
| `room` | 本会话 gg20 room |
| `participants` | 参与者列表 |
| `responses` | 参与状态 |
| `status` | 会话状态 |

#### sign_sessions

| 字段 | 说明 |
| --- | --- |
| `session_key` | 离线 sign 会话编号 |
| `task_no` | 关联任务编号 |
| `transaction_no` | 交易编号 |
| `from_address` | 签名地址 |
| `message_hash` | 待签名哈希 |
| `manager_addr` | 本会话 manager 地址 |
| `room` | 本会话 gg20 room |
| `participants` | 参与者列表 |
| `parties` | GG20 原始 shard index 列表，例如 `2,3` |
| `responses` | 参与状态 |
| `signature` | 最终签名 |
| `status` | 会话状态 |

#### audit_logs

| 字段 | 说明 |
| --- | --- |
| `username` | 操作人 |
| `role` | 操作角色 |
| `action` | 操作动作 |
| `resource_type` | 资源类型 |
| `resource_id` | 资源编号 |
| `result` | `success` / `failure` |
| `error_message` | 错误信息 |
| `sensitive_redacted` | 是否脱敏 |

审计日志不得记录明文 share、AES key、SE 授权签名、完整 `encrypted_shard`。

### 11.4 Keygen 流程

1. 协调员导入 `custody_keygen` 任务包。
2. 离线服务端校验包签名和 `payload_hash`。
3. 离线服务端创建 `offline_task`。
4. 协调员选择参与者和安全芯片。
5. 离线服务端创建 `keygen_session`。
6. 离线服务端计算 `gg20_threshold = required_signers - 1`。
7. 离线服务端启动独立 `gg20_sm_manager --address 0.0.0.0 --port <port>`。
8. 离线服务端生成 `manager_addr` 和唯一 `room`。
9. 离线服务端通过 WebSocket 邀请参与方。
10. 所有参与方接受后，下发 keygen 参数：
    - `manager_addr`
    - `room`
    - `gg20_threshold`
    - `total_parties`
    - `party_index`
    - `session_key`
11. 桌面端本地执行 `gg20_keygen`。
12. 桌面端生成随机 AES key，加密 local share。
13. 桌面端把 AES key 写入 SE。
14. 桌面端返回地址、公钥、CPLC、`encrypted_shard`。
15. 离线服务端保存 `key_shards`。
16. 所有参与方完成后，离线服务端生成 `custody_keygen_result` 结果包。
17. 离线服务端停止本会话 manager。

### 11.5 Sign 流程

1. 协调员导入 `sign` 任务包。
2. 离线服务端校验包签名和 `payload_hash`。
3. 离线服务端创建 `offline_task`。
4. 协调员选择参与签名的用户。
5. 离线服务端读取每个用户对应地址的 `key_shards`。
6. 离线服务端按选中用户得到原始 shard index 列表，生成 `parties`，例如 `2,3`。
7. 对每个参与方计算 `signing_index = position(shard_index in parties) + 1`。
8. 离线服务端创建 `sign_session`。
9. 离线服务端启动独立 `gg20_sm_manager`。
10. 离线服务端生成 `manager_addr` 和唯一 `room`。
11. 离线服务端邀请参与方。
12. 所有参与方接受后，下发 sign 参数：
    - `manager_addr`
    - `room`
    - `message_hash`
    - `parties`
    - `signing_index`
    - 当前参与方自己的 `encrypted_shard`
    - SE 读取授权签名
    - 交易展示字段
13. 桌面端要求参与者确认交易展示字段。
14. 桌面端用授权签名从 SE 读取 AES key。
15. 桌面端解密 `encrypted_shard`，写入临时 share 文件。
16. 桌面端执行 `gg20_signing --index <signing_index> --parties <parties>`。
17. 桌面端删除临时明文 share 文件。
18. 桌面端返回签名结果。
19. 离线服务端收集最终签名并生成 `sign_result` 结果包。
20. 离线服务端停止本会话 manager。

## 12. 离线桌面端设计

### 12.1 职责

离线桌面端负责：

- 操作员登录离线服务端。
- 接收 keygen/sign 邀请。
- 展示任务详情和交易内容。
- 调用本地 `gg20_keygen` / `gg20_signing`。
- 访问本机读卡器和 SE。
- 加密、解密本参与方 share。
- 返回 keygen/sign 结果。

离线桌面端不负责：

- 保存长期业务主数据。
- 管理全部参与方状态。
- 保存其他参与方 share。
- 查询链上信息。
- 广播交易。

### 12.2 组件

```text
offline-client-wails
  |
  +-- Vue UI
  |     +-- 任务确认
  |     +-- SE 状态
  |     +-- Keygen/Sign 操作
  |
  +-- Wails Go Backend
  |     +-- WebSocket client
  |     +-- MPCService
  |     +-- SecurityService
  |
  +-- Embedded GG20 Binaries
  |     +-- gg20_keygen_<os>_<arch>
  |     +-- gg20_signing_<os>_<arch>
  |
  +-- SE / Card Reader
        +-- Store AES key
        +-- Read AES key with authorization
        +-- Delete key material
```

### 12.3 SE 中保存什么

SE 不保存 GG20 local share 明文。推荐保存：

```text
record_key = hash(username || address || shard_index)
value = 32-byte AES-GCM key
metadata = address, shard_index, optional task/offline reference
```

离线服务端数据库保存：

```text
encrypted_shard = AES-GCM(local_share_json, AES key)
```

这样设计的原因：

- 服务端没有 AES key，不能解密 share。
- SE 没有完整 share，只保存解密材料。
- 参与者必须持有 SE 并经过授权，才能参与签名。
- 单独泄露服务端数据库或单独丢失 SE，都不足以恢复分片。

### 12.4 SE 读取授权签名

当前设计中，离线服务端会给桌面端一个签名，用来授权本次从 SE 读取 AES key。目标设计中，这个签名必须绑定具体操作上下文。

建议签名内容：

```json
{
  "operation": "sign",
  "task_no": "TASK-2026-0002",
  "session_key": "sign_...",
  "username_hash": "sha256:...",
  "address": "0x...",
  "shard_index": 2,
  "message_hash": "0x...",
  "nonce": "random-128-bit",
  "expires_at": "2026-05-28T12:50:00Z"
}
```

为什么需要绑定这些字段：

- 防止一次授权签名被重复使用。
- 防止把签名任务 A 的授权用于任务 B。
- 防止只凭 `username + address` 长期读取同一个 SE 记录。
- 让 SE 侧或客户端侧可以做过期校验和审计。

### 12.5 桌面端 Keygen 流程

1. 桌面端收到 `keygen_params`。
2. 校验 `manager_addr`、`room`、`party_index`、`gg20_threshold`、`total_parties`。
3. 根据当前平台释放或选择内嵌 `gg20_keygen`。
4. 使用本地临时目录创建唯一输出文件。
5. 执行：

```bash
gg20_keygen \
  --address <manager_addr> \
  --room <room> \
  --threshold <gg20_threshold> \
  --number-of-parties <total_parties> \
  --index <party_index> \
  --output <temp_share_file>
```

6. 读取 `temp_share_file`，提取地址和公钥。
7. 生成随机 32 字节 AES key。
8. 用 AES-GCM 加密 share JSON。
9. 将 AES key 写入 SE。
10. 删除临时 share 文件。
11. 返回 `address`、`public_key`、`cplc`、`encrypted_shard`。

### 12.6 桌面端 Sign 流程

1. 桌面端收到 `sign_params`。
2. 展示交易可读字段：地址、接收方、金额、费用上限、原因、消息哈希。
3. 用户确认后，校验 `parties` 和 `signing_index`。
4. 使用授权签名从 SE 读取 AES key。
5. 用 AES key 解密 `encrypted_shard`。
6. 将 share JSON 写入本地临时文件，权限应尽量限制为当前用户可读写。
7. 根据当前平台释放或选择内嵌 `gg20_signing`。
8. 执行：

```bash
gg20_signing \
  --address <manager_addr> \
  --room <room> \
  --index <signing_index> \
  --parties <parties> \
  --data-to-sign <message_hash_without_0x_or_expected_format> \
  --local-share <temp_share_file>
```

9. 删除临时 share 文件。
10. 转换为以太坊 RSV 签名格式。
11. 返回签名结果。

关键规则：

- keygen 的 `--index` 是原始 party index。
- signing 的 `--index` 是当前参与者在本次 `parties` 子集中的 1-based 位置。
- `parties` 是原始 shard index 列表，不能按当前参与者顺序重新生成 `1,2,3`。

## 13. 安全要求

- 日志不得输出 AES key。
- 日志不得输出明文 share。
- 日志不得输出完整 `encrypted_shard`。
- 临时 share 文件必须使用唯一文件名，流程结束后删除。
- keygen/sign 失败后也必须清理临时文件。
- `manager_addr` 和 `room` 必须按会话唯一。
- 每次 keygen/sign 建议独立启动 manager。
- 重试必须新建任务会话，不复用旧 manager、旧 room、旧临时文件。
- 在线结果导入必须验证签名和原任务一致性。
- 离线任务导入必须验证在线包签名和 `payload_hash`。

## 14. 当前代码改造清单

### 14.1 离线服务端

- 增加任务包导入和结果包导出 API。
- 增加 `offline_tasks` 和包哈希记录。
- 将全局 manager 改为会话级 manager。
- keygen 下发 `manager_addr` 和 `room`。
- sign 根据 `key_shards.shard_index` 生成 `parties`。
- sign 下发 `signing_index`，不要把 shard index 当 signing index。
- SE 授权签名绑定 task/session/message/nonce/expiry。
- 审计日志脱敏。

### 14.2 离线桌面端

- 替换为 `v0.8.1-gg20.3` 或更新版本的 GG20 二进制。
- `SignRequest` 增加 `signing_index`。
- `RunSigning` 增加 `--index` 参数。
- keygen 将业务门限转换为 GG20 `threshold = required_signers - 1`。
- 临时 share 文件改为本地生成唯一文件名。
- 删除 AES key 日志。
- 签名前展示交易可读字段，不只展示 hash。

### 14.3 在线系统

- 增加任务包下载能力。
- 增加结果包上传能力。
- 记录 `payload_hash`、`result_hash`、`offline_ref_no`。
- 导入 sign 结果后验证签名恢复地址。
- 在线数据库不增加任何密钥分片或 SE 明细字段。
