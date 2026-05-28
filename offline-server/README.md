# 离线协作服务端

`offline-server` 是离线端的协作服务，负责用户登录、角色权限、密钥生成会话、签名会话、安全芯片登记、WebSocket 实时通信和 MPC manager 进程管理。

## 核心能力

- 用户认证：提供注册、登录、登出和令牌校验。
- 角色权限：支持管理员、协调者和参与者等离线协作角色。
- 密钥生成：创建 MPC 密钥生成会话，并协调参与方完成流程。
- 签名协作：创建签名会话，通过 WebSocket 分发和收集签名消息。
- 安全芯片管理：登记安全芯片标识和 CPLC 信息。
- 进程管理：每个 keygen/sign 会话独立启动并清理 `gg20_sm_manager`。
- 本地存储：使用 SQLite 保存用户、会话、安全芯片和密钥分片相关数据。

## 目录结构

```text
offline-server/
├── bin/           # MPC manager 可执行文件
├── clog/          # 日志封装
├── docs/          # Web 和 WebSocket 模块文档
├── manager/       # 后台进程管理器
├── storage/       # SQLite 数据访问层
├── tools/         # JWT 等辅助工具
├── web/           # HTTP API 服务
├── ws/            # WebSocket 协调服务
├── go.mod         # Go 依赖定义
└── main.go        # 服务启动入口
```

## 服务端口

默认启动两个服务：

```text
HTTP API:  http://localhost:8080
WebSocket: ws://0.0.0.0:8081/ws
```

启动参数可调整端口：

```bash
go run . -web-port 8080 -ws-host 0.0.0.0 -ws-port 8081
```

## 本地运行

```bash
go mod tidy
go run .
```

运行前需要确认 `bin/` 下存在当前平台对应的 manager 二进制，并具有执行权限：

- macOS arm64：`gg20_sm_manager_darwin_arm64`
- Linux amd64：`gg20_sm_manager_linux_amd64`
- Windows amd64：`gg20_sm_manager_windows_amd64.exe`

启动后会自动创建 `data/` 和 `logs/` 目录。

会话级 manager 可通过环境变量配置：

```bash
# 可选；不设置时会按当前平台自动选择 ./bin/gg20_sm_manager_<goos>_<goarch>[.exe]
OFFLINE_MANAGER_BIN=./bin/gg20_sm_manager_linux_amd64
OFFLINE_MANAGER_BIND_ADDRESS=0.0.0.0
OFFLINE_MANAGER_PUBLIC_HOST=192.168.1.10
OFFLINE_MANAGER_PORT_START=18001
OFFLINE_MANAGER_PORT_END=18100
```

## Docker 部署

离线服务端镜像当前面向 Linux amd64，镜像内包含：

- `offline-server` 服务端可执行文件。
- `bin/gg20_sm_manager_linux_amd64`。

构建并推送镜像：

```bash
cd offline-server
./docker-build-push.sh ceyewan crypto-custody-offline-server latest
```

部署入口统一放在根目录的独立部署目录：

```bash
cd deploy/offline-server
cp .env.example .env
./deploy.sh
```

详细说明见 `deploy/offline-server/README.md`。镜像不会包含 `ec_private_key.pem`，部署时由 `deploy/offline-server/private_keys/` 挂载。

## 测试

```bash
go test ./...
```

当前测试以新 WebSocket 协议单元测试为主，不需要提前启动 HTTP/WebSocket 服务。

## 相关文档

- `docs/web_module_documentation.md`：HTTP API 模块说明。
- `docs/ws_module_documentation.md`：WebSocket 模块说明。
- `manager/README.md`：manager 进程管理器说明。
