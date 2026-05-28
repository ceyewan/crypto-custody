# ZenGo-X multi-party-ecdsa 使用摘要

本文记录 `ZenGo-X/multi-party-ecdsa` README 中与本项目离线系统相关的使用方式，以及本项目 fork 后需要固定下来的正确调用约定。源仓库：

https://github.com/ZenGo-X/multi-party-ecdsa

本项目使用的 fork：

https://github.com/ceyewan/multi-party-ecdsa

## 重要边界

- 该仓库已经不再维护，官方不会提供安全更新或热修复。项目使用时应将其视为可执行工具依赖，而不是安全托管服务。
- 该库实现的是门限 ECDSA，适合当前以太坊/secp256k1 交易签名方向。
- README 中的 GG20 demo 通信通道没有加密和认证；生产系统必须在外层做参与方身份校验、消息认证、会话隔离和审计。
- 本项目中，`offline-server` 生产环境主要部署 Linux 版本；CI 同时提供 Windows/Linux/macOS 的 `gg20_sm_manager`，方便本地和跨平台联调。桌面端 `offline-client-wails` 需要支持 Windows、Linux、macOS 三端运行。

## 编译

标准编译：

```bash
cargo build --release --examples
```

如果目标环境不方便安装 GMP，可使用纯 Rust 大整数后端：

```bash
cargo build --release --examples --no-default-features --features curv-kzen/num-bigint
```

编译产物位于：

```text
./target/release/examples/
```

本项目需要关注的 GG20 demo 二进制：

```text
gg20_sm_manager
gg20_keygen
gg20_signing
```

## GG20 运行模型

GG20 demo 由一个 SM server 协调多方通信：

```bash
./gg20_sm_manager
```

默认监听：

```text
http://127.0.0.1:8000
```

本项目 fork 支持显式指定监听地址和端口：

```bash
./gg20_sm_manager --address 0.0.0.0 --port 18001
```

如果参与方不在同一台机器，需要让所有参与方都能访问该 SM server，并在命令里显式指定地址：

```bash
./gg20_keygen --address http://10.0.1.9:8000/ ...
```

离线系统推荐每个 keygen/sign 会话启动一个独立 manager，分配一个独立端口，并把本次地址下发给所有参与方：

```json
{
  "manager_addr": "http://192.168.x.x:18001"
}
```

## Keygen 示例

README 的 GG20 示例生成的是 2-of-3 钱包。注意库里的 `t` 不是“需要几人签名”，而是阈值参数；实际需要 `t + 1` 个参与方签名。

2-of-3 对应：

```text
t = 1
n = 3
```

三个参与方分别执行：

```bash
./gg20_keygen -t 1 -n 3 -i 1 --output local-share1.json
./gg20_keygen -t 1 -n 3 -i 2 --output local-share2.json
./gg20_keygen -t 1 -n 3 -i 3 --output local-share3.json
```

执行完成后，每个参与方得到自己的本地分片文件：

```text
local-share1.json
local-share2.json
local-share3.json
```

这些文件是私密材料，不能进入在线系统。

## Signing 示例

2-of-3 钱包中任意两个参与方可签名。比如选择 1、2 号参与方：

```bash
./gg20_signing --index 1 -p 1,2 -d "hello" -l local-share1.json
./gg20_signing --index 2 -p 1,2 -d "hello" -l local-share2.json
```

参数含义：

- `-p 1,2`：本次参与签名的原始 party index 列表。
- `--index 1`：当前参与方在本次 `-p` 列表中的 1-based 位置。
- `-d "hello"`：待签名数据。
- `-l local-share1.json`：当前参与方自己的本地分片文件。

关键注意点：

- `-p` / `--parties` 必须使用 keygen 时分配的原始 party index。
- signing 的 `--index` 不是 keygen 原始 party index，而是当前参与方在 `-p` 列表中的 1-based 位置。
- 本地分片文件必须和 `-p` 中该位置对应的原始 party index 匹配。

例如 2-of-3 钱包选择 2、3 号参与方签名时，应使用：

```bash
./gg20_signing --index 1 -p 2,3 -d "hello" -l local-share2.json
./gg20_signing --index 2 -p 2,3 -d "hello" -l local-share3.json
```

这里 2 号分片虽然原始 party index 是 `2`，但它在本次 `-p 2,3` 中排第 1，所以 signing 时必须传 `--index 1`。错误示例：

```bash
./gg20_signing --index 2 -p 2,3 -d "hello" -l local-share2.json
```

这个命令会把 `local-share2.json` 当成本次签名子集里的第 2 个参与方处理，也就是和 `-p 2,3` 的 3 号位置冲突，可能导致签名流程卡住。

2-of-3 应覆盖的签名组合：

```bash
# 1,2
./gg20_signing --index 1 -p 1,2 -d "hello" -l local-share1.json
./gg20_signing --index 2 -p 1,2 -d "hello" -l local-share2.json

# 2,3
./gg20_signing --index 1 -p 2,3 -d "hello" -l local-share2.json
./gg20_signing --index 2 -p 2,3 -d "hello" -l local-share3.json

# 1,3
./gg20_signing --index 1 -p 1,3 -d "hello" -l local-share1.json
./gg20_signing --index 2 -p 1,3 -d "hello" -l local-share3.json

# 1,2,3
./gg20_signing --index 1 -p 1,2,3 -d "hello" -l local-share1.json
./gg20_signing --index 2 -p 1,2,3 -d "hello" -l local-share2.json
./gg20_signing --index 3 -p 1,2,3 -d "hello" -l local-share3.json
```

## 本项目推荐封装方式

离线系统不应让操作员直接执行上述命令，而应由 wrapper 统一封装：

- `offline-server` 负责会话创建、参与方邀请、参数下发、结果汇总、审计记录。
- `offline-client-wails` 负责在本机调用 `gg20_keygen` / `gg20_signing`，读取安全芯片，解密本地分片，返回结果。
- `gg20_sm_manager` 建议按会话启动，至少按 keygen/sign 任务隔离；失败或重试时关闭旧 manager，分配新端口和新会话。
- 服务端和本地测试可使用 Windows/Linux/macOS 对应的 `gg20_sm_manager`；客户端需要内置或随包分发 Windows/Linux/macOS 对应的 `gg20_keygen` 和 `gg20_signing`。

## 门限参数换算

业务口径通常说的是“几人可签”。GG20 命令里的 `threshold` 使用 `t`，实际签名人数是 `t + 1`。

| 业务口径 | GG20 参数 | 签名命令参与方数量 |
| --- | --- | --- |
| 2-of-2 | `-t 1 -n 2` | 2 |
| 2-of-3 | `-t 1 -n 3` | 2 |
| 3-of-3 | `-t 2 -n 3` | 3 |
| 3-of-5 | `-t 2 -n 5` | 3 |

因此代码中应使用：

```text
gg20Threshold = requiredSigners - 1
```

## 对当前实现的直接要求

- 不要把业务里的“2 人可签”直接传给 `--threshold 2`，否则会变成 3 人门限。
- 签名时 `parties` 必须来自分片保存时的 `ShardIndex`，不能按当前参与者顺序重新生成 `1,2,3`。
- 签名时传给 `gg20_signing --index` 的值必须是当前分片在 `parties` 中的位置，计算方式为 `position(ShardIndex in parties) + 1`，不能直接传 `ShardIndex`。
- 同一轮签名中，所有参与方必须使用完全一致且顺序一致的 `parties`。建议 wrapper 统一生成并下发，不要让各客户端自行排序或拼接。
- 参数下发时应使用唯一临时文件名，客户端不应写死 `keygen_temp.json` 或 `sign_temp.json`。
- `manager_addr` 不应长期固定为 `http://localhost:8000`；会话启动后应把本次 manager 地址下发给参与方。
- 重试应创建新 session、新 manager、新临时文件，避免复用上一次失败会话的状态。

## Smoke 测试

本项目 fork 中提供了 Go smoke 测试脚本，可用来验证下载后的二进制是否满足离线系统的基本要求：

```bash
# 在 ceyewan/multi-party-ecdsa 仓库根目录执行
go run scripts/gg20_smoke.go --bin-dir /path/to/extracted-binaries --port 18001 --iterations 10
```

该脚本默认会启动一个本地 `gg20_sm_manager`，使用 `--port` 指定端口，并在同一个 manager 下为每轮 keygen/sign 分配不同的 `--room`，避免测试会话互相串扰。生产封装仍应按上文要求为业务会话分配独立 manager 和端口。

`--bin-dir` 可以指向 release artifact 解压目录。脚本会优先查找当前系统对应的命名产物，例如 `gg20_keygen_darwin_arm64`、`gg20_signing_linux_amd64`、`gg20_sm_manager_windows_amd64.exe`，也兼容本地 `target/release/examples/` 下的未改名产物。

每轮测试会执行一次 3-party keygen，并覆盖 2-of-3 的 `1,2`、`2,3`、`1,3` 以及 `1,2,3` 签名组合。脚本里的 signing 命令按 `position(ShardIndex in parties) + 1` 计算 `--index`，并使用对应的 `local-shareN.json`。`--iterations 10` 可用于连续跑 10 轮，观察 GG20 demo 在当前机器和网络环境下是否稳定。
