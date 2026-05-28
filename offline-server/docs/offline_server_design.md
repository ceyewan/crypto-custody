# 离线服务端设计文档

## 1. 定位

离线服务端是离线系统的协作控制面，负责用户认证、会话协调、参与方邀请、任务状态记录、审计留痕和 `gg20_sm_manager` 生命周期管理。

离线服务端不直接持有私钥明文，不连接区块链网络，不负责在线案件主数据、账户批量导入、交易广播和在线备份恢复。

## 2. 部署边界

- 服务端只要求支持 Linux 部署。
- 桌面客户端才需要支持 Windows、Linux、macOS。
- 服务端需要随包部署 Linux 版本 `gg20_sm_manager`。
- `gg20_keygen` 和 `gg20_signing` 主要由桌面客户端本地执行。
- 服务端数据库使用本地 SQLite，记录离线用户、芯片登记信息、分片索引、会话和审计。

## 3. 核心组件

```text
offline-server
  |
  +-- Web API
  |     +-- 用户登录/注册/权限
  |     +-- 会话创建
  |     +-- 安全芯片登记
  |     +-- 任务包导入/结果包导出
  |
  +-- WebSocket Hub
  |     +-- 客户端注册
  |     +-- 邀请参与方
  |     +-- 下发 keygen/sign 参数
  |     +-- 收集结果
  |
  +-- Session Manager
  |     +-- keygen 会话
  |     +-- sign 会话
  |     +-- 超时/失败/重试状态
  |
  +-- SM Manager Runtime
  |     +-- 按会话启动 gg20_sm_manager
  |     +-- 分配端口
  |     +-- 结束后回收进程
  |
  +-- Storage
        +-- users
        +-- security_chips
        +-- key_shards
        +-- sessions
        +-- offline_tasks
        +-- audit_logs
```

## 4. 与在线系统的边界

在线系统导出任务包，离线服务端导入任务包并驱动离线流程。

在线到离线：

- 托管钱包生成任务：案件编号、币种、门限策略、任务编号。
- 签名任务：交易编号、地址、待签名哈希、业务说明。

离线到在线：

- 托管钱包生成结果：地址、公钥、任务编号、离线引用号。
- 签名结果：交易编号、消息哈希、签名值、完成时间。

离线结果包不得包含：

- 私钥。
- 助记词。
- local share 明文。
- 安全芯片内部数据。
- 参与者个人密钥材料。

## 5. 角色权限

| 角色 | 权限 |
| --- | --- |
| admin | 用户管理、安全芯片登记、系统配置、审计查询、异常处置 |
| coordinator | 导入任务包、发起 keygen/sign 会话、导出结果包 |
| participant | 接受会话邀请，使用桌面客户端完成本地 keygen/sign |
| auditor | 查询审计日志和操作证明 |

权限原则：

- keygen/sign 会话只能由 `admin` 或 `coordinator` 发起。
- 参与者只能处理分配给自己的会话。
- 安全芯片登记和状态变更需要 `admin`。
- 任何读取、删除、移交、销毁密钥材料的流程必须写审计。

## 6. GG20 会话设计

本项目使用 `ZenGo-X/multi-party-ecdsa` 的 GG20 demo 二进制：

- `gg20_sm_manager`：多方通信协调服务。
- `gg20_keygen`：参与方本地执行密钥生成。
- `gg20_signing`：参与方本地执行签名。

推荐设计是**每个 keygen/sign 会话独立启动一个 `gg20_sm_manager`**。

原因：

- demo manager 不适合作为多任务共享的长期并发服务。
- 失败重试时需要隔离上一轮连接和状态。
- 多案件、多签名任务同时操作时，独立端口能降低串线风险。
- 会话结束后可以直接停止进程，清理更明确。

会话运行时字段：

| 字段 | 说明 |
| --- | --- |
| session_key | 离线会话编号 |
| task_no | 在线任务编号 |
| session_type | keygen/sign |
| manager_port | 本会话 manager 端口 |
| manager_addr | 本会话 manager 地址 |
| participants | 参与者用户名列表 |
| party_indexes | 原始 party index 列表 |
| deadline_at | 会话截止时间 |
| status | created/invited/accepted/processing/completed/failed/cancelled |

## 7. 门限参数换算

业务上说的“几人可签”不是 GG20 的 `threshold` 原值。GG20 的签名参与人数是 `t + 1`。

| 业务口径 | GG20 keygen 参数 | 签名参与人数 |
| --- | --- | --- |
| 2-of-2 | `-t 1 -n 2` | 2 |
| 2-of-3 | `-t 1 -n 3` | 2 |
| 3-of-3 | `-t 2 -n 3` | 3 |
| 3-of-5 | `-t 2 -n 5` | 3 |

代码中应统一使用：

```text
gg20Threshold = requiredSigners - 1
```

## 8. Keygen 流程

1. 协调员导入在线托管钱包生成任务包。
2. 离线服务端创建 keygen 会话。
3. 服务端选择空闲端口并启动本会话 `gg20_sm_manager`。
4. 协调员选择参与者和门限策略。
5. 服务端通过 WebSocket 邀请参与者。
6. 所有参与者接受后，服务端下发：
	   - `manager_addr`
	   - `room`
	   - `threshold`
	   - `total_parties`
	   - `party_index`
	   - `record_id`
	   - 唯一临时文件名
7. 各桌面客户端本地运行 `gg20_keygen`。
8. 客户端提取地址和 local share。
9. 客户端生成随机 32 字节密钥，加密 local share。
10. 客户端将随机密钥写入安全芯片。
11. 客户端把加密后的 share 返回服务端保存。
12. 服务端确认所有参与方完成后，导出托管钱包结果包。
13. 服务端停止本会话 `gg20_sm_manager`。

## 9. Signing 流程

1. 协调员导入在线签名任务包。
2. 离线服务端创建 sign 会话。
3. 服务端选择空闲端口并启动本会话 `gg20_sm_manager`。
4. 协调员选择参与者。
5. 服务端根据已保存的 `shard_index` 生成 `parties` 参数。
6. 服务端通过 WebSocket 邀请参与者。
7. 所有参与者接受后，服务端下发：
	   - `manager_addr`
	   - `room`
	   - `message_hash`
	   - `parties`
	   - 当前参与者自己的 `party_index`
	   - 当前参与者自己的 `signing_index`
	   - 当前参与者自己的 `record_id`
	   - 当前参与者自己的加密 share
   - 安全芯片读取授权签名
   - 唯一临时文件名
8. 客户端从安全芯片读取解密密钥。
9. 客户端解密 local share，写入临时文件。
10. 客户端运行 `gg20_signing`。
11. 客户端删除临时明文 share 文件。
12. 客户端返回签名结果。
13. 服务端汇总完成后导出签名结果包。
14. 服务端停止本会话 `gg20_sm_manager`。

关键要求：

- `parties` 必须使用 keygen 时的原始 `shard_index`，不能按本次参与者顺序重新编号。
- 客户端必须使用服务端下发的唯一临时文件名，不能写死 `keygen_temp.json` 或 `sign_temp.json`。
- 重试必须创建新 session、新 manager、新临时文件。

## 10. 数据模型

### User

| 字段 | 说明 |
| --- | --- |
| username | 用户名 |
| password_hash | 密码哈希 |
| role | admin/coordinator/participant/auditor |
| status | active/disabled |

### SecurityChip

| 字段 | 说明 |
| --- | --- |
| se_id | 芯片业务编号 |
| cplc | 芯片唯一标识 |
| owner | 当前持有人 |
| status | active/lost/disabled/destroyed |
| registered_by | 登记人 |

### KeyShard

| 字段 | 说明 |
| --- | --- |
| address | 托管钱包地址 |
| username | 分片持有人 |
| shard_index | keygen 时的原始 party index |
| cplc | 对应安全芯片 |
| encrypted_shard | 加密后的 local share |
| status | active/transferred/destroyed |

### OfflineTask

| 字段 | 说明 |
| --- | --- |
| task_no | 任务编号 |
| task_type | custody_keygen/sign/private_key_transfer/private_key_destroy |
| payload_hash | 导入任务包哈希 |
| result_hash | 导出结果包哈希 |
| status | imported/processing/completed/failed |

### Session

| 字段 | 说明 |
| --- | --- |
| session_key | 会话编号 |
| task_no | 关联任务编号 |
| session_type | keygen/sign |
| manager_addr | 本会话 SM manager 地址 |
| participants | 参与者 |
| status | 会话状态 |

### AuditLog

| 字段 | 说明 |
| --- | --- |
| username | 操作人 |
| action | 操作 |
| resource_type | 资源类型 |
| resource_id | 资源编号 |
| result | success/failure |
| error_message | 错误信息 |
| created_at | 时间 |

## 11. 异常和重试

| 场景 | 处理 |
| --- | --- |
| 参与者拒绝 | 会话失败，记录原因，可重新创建会话 |
| 参与者超时 | 会话失败，停止 manager，允许重试 |
| manager 端口占用 | 换端口启动，记录异常 |
| manager 进程退出 | 会话失败，停止并清理状态 |
| 客户端断线 | 会话失败或等待重连，按超时策略处理 |
| 签名失败 | 返回失败原因，重试必须新建会话 |
| 安全芯片读取失败 | 会话失败，记录芯片和参与者，不泄露密钥信息 |

## 12. 当前实现差距

当前实现已经具备：

- 用户、角色和基础认证。
- WebSocket 邀请、响应、参数、结果消息。
- keygen/sign 会话内存管理。
- Linux 版全局 `gg20_sm_manager` 启动。
- 安全芯片登记和 CPLC 校验。
- 桌面端本地执行 `gg20_keygen` / `gg20_signing`。

仍需调整：

- 从全局单个 `gg20_sm_manager` 改为会话级 manager。
- keygen 门限参数应从业务人数转换为 `requiredSigners - 1`。
- signing 的 `parties` 应使用原始 `shard_index`。
- 参数消息中增加 `manager_addr`，客户端按会话连接。
- 客户端使用服务端下发的临时文件名。
- 补齐 `OfflineTask` 和 `AuditLog`。
- 补齐任务包导入、结果包导出、会话重试和超时控制。
