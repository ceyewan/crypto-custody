# 离线存管提控系统详细设计

## 1. 目标和范围

本文档定义离线系统在虚拟货币存管提控原型中的目标设计，覆盖测试大纲中与离线密钥、安全芯片、MPC 门限签名、私钥查询、私钥移交、私钥销毁和提取控制相关的能力。

设计目标是：

- 能在离线环境中保存和使用密钥材料。
- 能通过安全芯片控制密钥材料读取。
- 能通过 MPC 生成托管钱包地址并对在线系统导入的交易哈希签名。
- 能支持私钥或密钥控制权的查询、移交、销毁和审计。
- 能与在线系统通过 JSON 任务包和结果包交互。
- 不把在线案件、链上查询、交易广播、备份恢复等职责搬到离线系统里。

本文档是实现口径，不追求完整密码基础设施。原型阶段优先满足存管提控闭环和测试大纲功能项。

## 2. 测试大纲口径

测试大纲中 2-1 到 2-8、2-12 与离线系统强相关；2-9、2-10、2-11、2-13 主要由在线系统完成，离线系统只参与离线签名和任务交互。

| 测试项 | 离线系统设计口径 |
| --- | --- |
| 2-1 离线私钥存储 | 离线服务端保存加密密钥材料，SE 保存解密 key；明文只在桌面端临时存在 |
| 2-2 私钥信息查询 | 授权后查询密钥记录、地址、状态、SE 绑定和脱敏摘要，不默认导出明文私钥 |
| 2-3 私钥移交 | 统一管理 SE 时优先做服务器元数据移交；如换 SE，则执行读旧 SE、写新 SE、验证、删除旧记录 |
| 2-4 私钥销毁 | 删除 SE 内解密 key，并把服务器密钥记录标记为 destroyed，使密文不可恢复 |
| 2-5 门限密钥生成 | 通过 MPC keygen 生成托管钱包地址、公钥和各参与方 share |
| 2-6 门限签名生成 | 由满足门限的参与方读取 SE、解密 share、执行 MPC sign |
| 2-7 门限签名验证 | 在线导入签名后验证签名与地址、哈希和交易一致 |
| 2-8 门限签名性能 | 提供 smoke/perf 工具记录标准签名和门限签名耗时 |
| 2-12 提取控制 | 在线创建交易并导出签名任务；离线签名；在线导入签名并广播 |

当前代码和 GG20 fork 支持的是 GG20/ECDSA/Ethereum 原型。测试大纲文字要求“国密 SM2”。如果验收必须严格 SM2，需要替换或补充 MPC 引擎，使同一套任务包、会话、SE 和审计流程调用 SM2 门限签名实现。本文档在数据结构中保留 `algorithm` 字段，但原型实现可以先使用 `GG20_ECDSA_SECP256K1`。

## 3. 总体架构

```text
在线系统
  |
  |  JSON 任务包 / JSON 结果包
  v
离线服务端
  |
  +-- HTTP API
  +-- WebSocket 协调
  +-- SQLite 数据库
  +-- 会话级 gg20_sm_manager
  |
  v
离线桌面端
  |
  +-- Wails UI
  +-- 本地 gg20_keygen / gg20_signing
  +-- PC/SC 读卡器
  +-- 安全芯片 Applet
```

职责划分：

| 组件 | 职责 |
| --- | --- |
| 在线系统 | 案件、账户、交易申请、链上查询、交易构建、结果导入、广播 |
| 离线服务端 | 任务导入、用户认证、SE 管理、MPC 会话协调、加密 share 保存、审计 |
| 离线桌面端 | 参与方操作、SE 读写、本地 MPC 二进制执行、临时明文清理 |
| SE | 保存 32 字节解密 key，校验读取/删除授权 |
| MPC manager | 单个 keygen/sign 会话内的多方消息转发 |

在线系统不能接触：

- 明文私钥。
- 明文 MPC share。
- 加密 share。
- SE 内 AES key。
- SE 读取授权签名。
- 参与方和 SE 的详细映射。

离线系统不能负责：

- 链上 nonce、gas、余额查询。
- 交易广播。
- 在线案件主数据。
- 在线系统热备份和冷备份。

## 4. 核心密钥模型

原型阶段统一使用“服务器密文 + SE 解密 key”的双因子模型。

```text
离线服务端:
  encrypted_blob = AES-GCM(secret_material, aes_key)

安全芯片:
  record_id + address -> aes_key
```

其中 `secret_material` 可以是：

- GG20 local share JSON。
- 测试或导入场景下的收缴私钥密文对象。

SE 不保存完整私钥，也不保存完整 GG20 share。服务端不保存 AES key。单独拿到服务端数据库或单独拿到 SE，都不能恢复密钥材料。

### 4.1 record_id

`record_id` 是 SE 内一条密钥记录的稳定编号，不是用户名，也不是 owner。

建议生成规则：

```text
record_id = sha256("offline-secret:v1|" + offline_key_id + "|" + shard_index + "|" + key_version)
```

当前 Applet 的第一个字段叫 `userName`，长度 32 字节；上层协议统一把该字段当作 `record_id` 使用。

SE 记录键：

```text
record_id(32 bytes) + address(20 bytes)
```

服务器保存：

```text
offline_key_id
address
shard_index
key_version
record_id
se_cplc
logical_owner
status
```

这样归属从 A 转到 B 时，只改 `logical_owner` 和权限关系，`record_id` 不变，SE 内数据不需要迁移。

### 4.2 SE 统一管理

SE 不归属个人，而是由离线系统统一管理。用户在 keygen 或 sign 时领取或插入指定 SE 参与操作。

SE 的业务含义：

```text
se_id: 人可读编号，例如 SE-001
cplc: 芯片唯一标识
status: active / lost / disabled / destroyed
custody_location: 保管位置
```

不建议把 SE 设计成“某个人永远拥有”。人员归属、案件归属、审批权限都放在服务器数据库中。

## 5. 角色、认证和权限

### 5.1 角色

| 角色 | 权限 |
| --- | --- |
| admin | 用户管理、SE 登记、系统配置、销毁确认、审计查询 |
| coordinator | 导入任务包、发起 keygen/sign、选择参与方、导出结果包 |
| participant | 接受会话邀请，插入指定 SE，执行本地 keygen/sign |
| auditor | 查询审计日志，参与移交/销毁复核 |

### 5.2 登录认证

原型阶段使用账号密码登录即可：

- 密码使用 bcrypt 或 argon2id 保存。
- 登录后颁发离线服务端 session/JWT。
- HTTP API 和 WebSocket 都必须校验登录态。
- WebSocket 注册时绑定用户名、角色和当前桌面端连接。
- 连续失败登录需要限速或锁定。

### 5.3 操作授权

关键操作必须校验角色：

| 操作 | 最低权限 |
| --- | --- |
| 导入任务包 | coordinator |
| 发起 keygen/sign | coordinator |
| 登记和停用 SE | admin |
| 私钥查询 | admin 或 coordinator，且需要审计 |
| 私钥移交 | admin 发起，auditor 复核 |
| 私钥销毁 | admin 发起，auditor 复核 |
| 导出结果包 | coordinator |

原型阶段的审批可以用“登录用户二次确认 + 审批记录”实现，不需要额外引入复杂 CA。后续如验收要求签名链，可把审批记录升级为离线系统操作签名。

## 6. 离线服务端设计

### 6.1 组件

```text
offline-server
  |
  +-- auth service
  +-- task package service
  +-- SE registry service
  +-- key/shard service
  +-- keygen session service
  +-- sign session service
  +-- manager runtime
  +-- audit service
  +-- SQLite storage
```

服务端只需要 Linux 部署。桌面端负责跨平台。

### 6.2 数据库

#### users

| 字段 | 说明 |
| --- | --- |
| id | 主键 |
| username | 用户名 |
| password_hash | 密码哈希 |
| role | admin/coordinator/participant/auditor |
| status | active/disabled |
| created_at/updated_at | 时间 |

#### security_chips

| 字段 | 说明 |
| --- | --- |
| id | 主键 |
| se_id | 人可读芯片编号 |
| cplc | 芯片唯一标识 |
| status | active/lost/disabled/destroyed |
| custody_location | 保管位置 |
| registered_by | 登记人 |
| created_at/updated_at | 时间 |

#### offline_keys

记录一个离线托管密钥或导入私钥对象。

| 字段 | 说明 |
| --- | --- |
| offline_key_id | 离线密钥编号 |
| case_no | 案件编号 |
| address | 地址 |
| coin_type | ETH/SM2_TEST 等 |
| algorithm | GG20_ECDSA_SECP256K1 / THRESHOLD_SM2 |
| threshold_policy | 例如 2-of-3 |
| public_key | 公钥 |
| logical_owner | 业务归属人或归属单位 |
| status | active/transferred/destroyed |

#### key_shards

记录每个 MPC 分片或导入秘密的密文。

| 字段 | 说明 |
| --- | --- |
| shard_id | 分片编号 |
| offline_key_id | 所属密钥 |
| address | 地址 |
| shard_index | keygen 原始 party index |
| record_id | SE 记录编号 |
| se_cplc | 绑定 SE |
| encrypted_blob | AES-GCM 密文 |
| blob_type | mpc_share/imported_private_key |
| key_version | 密钥版本 |
| status | active/transferred/destroyed |

#### offline_tasks

| 字段 | 说明 |
| --- | --- |
| task_no | 在线任务编号 |
| task_type | custody_keygen/sign/transfer/destroy |
| source_system | online/manual/test |
| payload_hash | 导入 payload 哈希 |
| result_hash | 导出 payload 哈希 |
| raw_package_path | 原任务包归档路径，可选 |
| status | imported/processing/completed/failed |

#### keygen_sessions

| 字段 | 说明 |
| --- | --- |
| session_key | 离线会话编号 |
| task_no | 任务编号 |
| required_signers | 业务门限人数 |
| total_parties | 总参与方数 |
| gg20_threshold | `required_signers - 1` |
| manager_addr | 本会话 manager 地址 |
| room | 本会话 room |
| participants | 参与者 |
| chips | 本会话指定 SE |
| status | created/invited/processing/completed/failed |

#### sign_sessions

| 字段 | 说明 |
| --- | --- |
| session_key | 离线会话编号 |
| task_no | 任务编号 |
| transaction_no | 交易编号 |
| from_address | 签名地址 |
| message_hash | 待签名哈希 |
| manager_addr | 本会话 manager 地址 |
| room | 本会话 room |
| participants | 本次签名参与者 |
| parties | 原始 shard index 列表，例如 `1,3` |
| status | created/invited/processing/completed/failed |
| signature | 最终签名 |

#### approvals

| 字段 | 说明 |
| --- | --- |
| approval_id | 审批编号 |
| operation | transfer/destroy/export_sensitive |
| resource_id | offline_key_id 或 shard_id |
| requested_by | 发起人 |
| approved_by | 审批人 |
| role | 审批人角色 |
| status | pending/approved/rejected |

#### audit_logs

| 字段 | 说明 |
| --- | --- |
| username | 操作人 |
| role | 操作角色 |
| action | 操作 |
| resource_type | 资源类型 |
| resource_id | 资源编号 |
| result | success/failure |
| error_message | 错误原因 |
| redacted_detail | 脱敏详情 |
| created_at | 时间 |

审计日志不得记录：

- AES key。
- 明文 share。
- 明文私钥。
- SE 授权签名。
- 完整 `encrypted_blob`。

### 6.3 API

原型阶段保留简单 API：

| API | 方法 | 说明 |
| --- | --- | --- |
| `/api/auth/login` | POST | 登录 |
| `/api/users` | CRUD | 用户管理 |
| `/api/se` | CRUD | SE 登记和状态管理 |
| `/api/offline/tasks/import` | POST | 导入在线任务包 |
| `/api/offline/results/{task_no}/download` | GET | 下载离线结果包 |
| `/api/keygen/start` | POST | 发起 keygen 会话 |
| `/api/sign/start` | POST | 发起 sign 会话 |
| `/api/keys/{offline_key_id}` | GET | 查询密钥脱敏信息 |
| `/api/keys/{offline_key_id}/transfer` | POST | 发起归属或控制权移交 |
| `/api/keys/{offline_key_id}/destroy` | POST | 发起销毁 |
| `/api/audit` | GET | 查询审计 |

WebSocket 用于：

- 桌面端注册连接。
- 下发 keygen/sign 邀请。
- 下发 keygen/sign 参数。
- 收集参与者结果。
- 推送会话状态。

## 7. 任务包和结果包

在线和离线之间只交换 JSON 文件。格式以 [online_offline_mpc_contract.md](online_offline_mpc_contract.md) 为准。

关键规则：

- 所有包包含 `schema_version`、`package_type`、`task_type`、`task_no`、`payload`、`payload_hash`。
- 原型阶段可以先实现 `payload_hash`，系统级包签名字段保留。
- 任务包不得包含离线参与方和 SE 明细。
- 结果包不得包含 share、密文 share、SE 授权或 CPLC 明细。

签名任务包必须包含可读展示字段：

```json
{
  "transaction_no": "TX-2026-0001",
  "from_address": "0x...",
  "to_address": "0x...",
  "value": "0.01",
  "message_hash": "0x...",
  "reason": "案件资金提取",
  "display": {
    "asset": "ETH",
    "amount": "0.01",
    "recipient_label": "案件资金接收账户",
    "fee_limit": "0.001"
  }
}
```

桌面端签名前必须展示这些字段，不能只展示 hash。

## 8. MPC 设计

### 8.1 引擎口径

当前实现使用：

```text
GG20_ECDSA_SECP256K1
```

对应二进制：

```text
gg20_sm_manager
gg20_keygen
gg20_signing
```

如果后续严格要求 SM2，保持服务端和桌面端业务流程不变，只替换 MPC 引擎：

```text
threshold_sm2_manager
threshold_sm2_keygen
threshold_sm2_signing
```

任务包中的 `algorithm` 用于选择引擎。

### 8.2 manager

每次 keygen 或 sign 都启动独立 manager：

```bash
gg20_sm_manager --address 0.0.0.0 --port <free_port>
```

原因：

- 会话隔离清晰。
- 失败后直接杀进程，避免旧状态污染。
- 多个会话可以用不同端口并发。
- 跨平台桌面端只需要访问离线服务器上的 manager 地址。

会话结束、失败或超时后，服务端必须停止对应 manager。

### 8.3 keygen 参数

业务门限 `required_signers` 和 GG20 `threshold` 不一样：

```text
gg20_threshold = required_signers - 1
```

例如 2-of-3：

```text
required_signers = 2
total_parties = 3
gg20_threshold = 1
```

每个参与者收到：

```text
manager_addr
room
gg20_threshold
total_parties
party_index
session_key
```

`party_index` 是 keygen 时的原始分片编号，必须保存到 `key_shards.shard_index`。

### 8.4 sign 参数

签名时可以由任意满足门限的分片参与，例如 2-of-3 可测试：

```text
1,2
1,3
2,3
1,2,3
```

`parties` 必须使用 keygen 时保存的原始 `shard_index` 列表，不能按本次参与者顺序重新编号。

GG20 signing 的 `--index` 是当前参与者在本次 `parties` 子集中的 1-based 位置：

```text
parties = 2,3
shard_index 2 -> signing_index 1
shard_index 3 -> signing_index 2
```

每个参与者收到：

```text
manager_addr
room
message_hash
parties
signing_index
encrypted_blob
record_id
address
se_authorization_signature
display
```

## 9. SE 设计

### 9.1 当前 Applet 能力

当前 Applet 支持：

| APDU | 说明 |
| --- | --- |
| STORE_DATA | 写入 `record_id + address -> 32-byte AES key` |
| READ_DATA | 读取 AES key，需要 ECDSA 授权签名 |
| DELETE_DATA | 删除 AES key，需要 ECDSA 授权签名 |

现有 Applet 字段名仍是 `userName`，上层语义固定为 `record_id`。

### 9.2 SE 写入

keygen 完成后：

1. 桌面端生成随机 32 字节 AES key。
2. 用 AES-GCM 加密 local share。
3. 计算或接收 `record_id`。
4. 调用 `STORE_DATA(record_id, address, aes_key)`。
5. 把 `encrypted_blob`、`record_id`、`cplc`、`shard_index` 返回服务端。
6. 删除本地明文 share 文件。

### 9.3 SE 读取

sign 时：

1. 服务端校验用户、SE 状态、会话、任务和审批。
2. 服务端生成 SE 读取授权签名。
3. 桌面端调用 `READ_DATA(record_id, address, signature)`。
4. 读出 AES key 后只在内存中使用。
5. 解密 local share，执行签名。
6. 清理 AES key 和明文 share。

当前 Applet 校验签名内容为：

```text
record_id || address
```

服务端必须在业务层限制授权签名的用途、有效期和会话。Applet v2 再考虑把 `operation`、`session_key`、`nonce`、`expires_at` 放入芯片验签内容。

### 9.4 SE 删除

销毁或迁移完成后：

1. 服务端生成删除授权签名。
2. 桌面端调用 `DELETE_DATA(record_id, address, signature)`。
3. 桌面端再尝试 READ，预期返回不存在。
4. 服务端把 `key_shards.status` 标记为 `destroyed` 或 `transferred`。

## 10. 桌面端设计

桌面端基于 Wails：

```text
Vue UI
  |
Wails Go Backend
  |
  +-- WebSocket client
  +-- MPCService
  +-- SecurityService
  +-- embedded binaries
  +-- PC/SC reader
```

### 10.1 跨平台二进制

客户端随包携带：

```text
gg20_keygen_darwin_arm64
gg20_signing_darwin_arm64
gg20_keygen_linux_amd64
gg20_signing_linux_amd64
gg20_keygen_windows_amd64.exe
gg20_signing_windows_amd64.exe
```

用户运行桌面端不需要安装 Rust。只需要：

- 桌面应用本身。
- 读卡器驱动和 PC/SC 服务。
- 可访问离线服务端和本会话 manager 的网络。
- 对应平台的可执行文件权限。

### 10.2 keygen 客户端流程

1. 收到 keygen 邀请。
2. 展示任务、门限、参与者、SE 要求。
3. 用户插入指定 SE。
4. 读取 CPLC 并上报服务端校验。
5. 执行 `gg20_keygen`。
6. 生成 AES key，加密 local share。
7. 写入 SE。
8. 上传 `encrypted_blob` 和摘要。
9. 删除临时明文文件。

### 10.3 sign 客户端流程

1. 收到 sign 邀请。
2. 展示交易可读字段。
3. 用户确认。
4. 读取 CPLC 并上报服务端校验。
5. 用授权签名从 SE 读取 AES key。
6. 解密 share 到本地临时文件。
7. 执行 `gg20_signing`。
8. 转换签名格式。
9. 删除临时明文 share 文件。
10. 返回签名。

桌面端禁止：

- 持久保存明文 share。
- 日志输出 AES key。
- 日志输出完整 share。
- 在 UI 上展示密钥明文。

## 11. 私钥查询、移交和销毁

### 11.1 查询

默认查询的是密钥记录状态，不是明文私钥：

```text
offline_key_id
address
algorithm
threshold_policy
logical_owner
shard_count
active_shard_count
SE 状态摘要
last_used_at
status
```

如测试需要“授权查询私钥信息”，原型口径是返回加密密钥材料摘要或经授权导出的密文，不直接显示明文私钥。明文只允许在桌面端内存中短暂出现，并且必须写审计。

### 11.2 归属权移交

SE 统一管理时，A 转给 B 的常规流程：

1. admin 发起移交。
2. auditor 复核。
3. 服务端校验 `offline_key_id` 状态为 active。
4. 更新 `logical_owner = B`。
5. 更新授权用户或使用策略。
6. 写审计日志和移交结果包。

SE 内记录不变，`record_id` 不变，`encrypted_blob` 不变。

### 11.3 控制权迁移到新 SE

如果必须从旧 SE 迁移到新 SE：

1. 校验旧 SE 和新 SE 均已登记。
2. 旧 SE 授权 READ，桌面端读出 AES key。
3. 新 SE 执行 STORE，写入相同 AES key 或新版本 AES key。
4. 新 SE 执行 READ 验证可读。
5. 旧 SE 执行 DELETE。
6. 服务端更新 `se_cplc`、`record_id` 或 `key_version`。
7. 写审计日志。

### 11.4 销毁

销毁流程：

1. admin 发起销毁。
2. auditor 复核。
3. 服务端生成删除授权。
4. 桌面端对相关 SE 执行 DELETE。
5. 桌面端验证 READ 失败。
6. 服务端标记 `key_shards.status = destroyed`。
7. 如果该密钥已无可用分片，标记 `offline_keys.status = destroyed`。
8. 导出销毁证明摘要。

销毁后可以保留 `encrypted_blob` 用于审计，但因为 SE 内 AES key 已删除，无法恢复明文。

## 12. 提取控制流程

完整闭环：

1. 在线系统中创建提取交易申请。
2. 在线系统完成审批、nonce/gas/余额查询和 unsigned transaction 构建。
3. 在线系统导出 `sign` 任务包。
4. 离线协调员导入任务包。
5. 离线服务端校验 `payload_hash`。
6. 协调员选择满足门限的参与方和 SE。
7. 离线服务端启动会话级 manager。
8. 桌面端展示交易详情，用户确认。
9. 参与者读取 SE、解密 share、执行 MPC 签名。
10. 离线服务端汇总签名并导出 `sign_result`。
11. 在线系统导入结果包，校验任务编号、交易编号、哈希和签名恢复地址。
12. 在线系统广播交易并更新状态。
13. 在线和离线各自写审计日志。

异常阻断：

| 场景 | 处理 |
| --- | --- |
| 签名人数不足 | 不启动签名或会话失败 |
| SE 不匹配 | 当前参与者失败，记录审计 |
| 交易展示字段被篡改 | payload_hash 校验失败，拒绝导入 |
| 签名恢复地址不匹配 | 在线系统拒绝导入或广播 |
| 用户拒绝确认 | 会话失败或等待更换参与方 |

## 13. 安全要求

### 13.1 数据安全

- 服务端只保存 `encrypted_blob`。
- SE 只保存 32 字节 AES key。
- 临时明文 share 文件必须使用唯一文件名。
- keygen/sign 成功、失败、取消都要清理临时文件。
- 日志和审计必须脱敏。
- SQLite 文件建议部署在磁盘加密分区。

### 13.2 网络安全

- 离线服务端运行在隔离网络。
- 桌面端只连接离线服务端和本次 manager。
- manager 端口只在会话期间开放。
- 会话结束后关闭 manager。
- WebSocket 必须校验登录态。

### 13.3 授权安全

- SE READ/DELETE 必须有服务端授权签名。
- 授权签名由离线服务端私钥生成。
- 原型阶段 Applet 校验 `record_id || address`。
- 服务端业务层必须把授权绑定到 `session_key`、`task_no`、用户、SE、有效期。
- 授权签名不得落日志，不得进入在线系统。

### 13.4 审计

必须审计：

- 登录成功和失败。
- 任务包导入和结果包导出。
- keygen/sign 会话创建、邀请、完成、失败。
- SE 登记、停用、挂失、销毁。
- 私钥查询、移交、销毁。
- 签名结果生成。

审计记录必须能回答：

```text
谁在什么时间，因为哪个任务，对哪个地址，使用哪些 SE，执行了什么操作，结果是什么。
```

## 14. 部署和运行

### 14.1 离线服务端

部署目标：

```text
Ubuntu 24.04 LTS
```

运行依赖：

- offline-server 可执行文件。
- SQLite 数据库目录。
- `gg20_sm_manager_linux_amd64`。
- 离线系统授权签名私钥。

服务端不需要安装 Rust。

### 14.2 离线桌面端

部署目标：

```text
Windows 10/11
macOS arm64
Linux amd64
```

运行依赖：

- 桌面应用。
- 对应平台 GG20 keygen/sign 二进制。
- PC/SC 读卡器驱动。
- 安全芯片 Applet。

桌面端不需要安装 Rust 或 Go。

## 15. 测试设计

### 15.1 MPC smoke

2-of-3 至少覆盖：

```text
1,2
1,3
2,3
1,2,3
```

每组支持连续运行 10 次，用于发现 manager、parties、signing_index 和临时文件清理问题。

### 15.2 SE 测试

覆盖：

- STORE 后 READ 成功。
- 无效签名 READ 失败。
- DELETE 后 READ 失败。
- record_id 不随 owner 变化。
- SE 状态 lost/disabled/destroyed 后拒绝会话。

### 15.3 提取控制测试

覆盖：

- 在线导出签名任务。
- 离线导入并签名。
- 在线导入签名并验签。
- 签名人数不足被拒绝。
- 篡改 `message_hash` 被拒绝。

### 15.4 性能测试

记录：

- keygen 总耗时。
- sign 总耗时。
- SE READ 耗时。
- 标准签名耗时。
- 门限签名耗时。

如果严格按照测试大纲验证 SM2 `t2/t1 <= 2`，必须使用同一 SM2 标准签名实现和 SM2 门限签名实现测试。当前 GG20/Ethereum 原型只能给出 ECDSA 门限签名性能数据。

## 16. 当前代码改造清单

为了落到本文档，当前代码需要优先改：

1. SE 上层参数统一使用 `record_id`。
2. `key_shards` 使用 `record_id`、`key_version`、`status`，统一 CPLC 命名。
3. 增加 `offline_tasks`、`offline_keys`、`approvals`、`audit_logs`。
4. 增加任务包导入和结果包导出 API。
5. keygen/sign 改为每个会话独立 manager。
6. sign 下发 `parties` 和 `signing_index`，并按原始 `shard_index` 计算。
7. 桌面端签名前展示交易可读字段。
8. 删除 AES key、share、授权签名相关日志。
9. 补齐私钥查询、归属移交、SE 迁移和销毁流程。
10. 如果验收强制 SM2，增加或替换 MPC 引擎。

## 17. 最小可验收版本

最小可验收版本不做复杂扩展，只需要达到：

- 3 个离线用户、3 个 SE、2-of-3 keygen 成功。
- 2-of-3 的任意两方和三方签名均成功。
- 每次 keygen/sign 使用独立 manager。
- SE 保存 AES key，服务端保存加密 share。
- 归属权移交只改服务器元数据，SE 不变。
- 销毁能删除 SE 记录，删除后无法读取。
- 在线和离线通过 JSON 任务包/结果包完成提取控制闭环。
- 所有敏感操作有脱敏审计日志。
