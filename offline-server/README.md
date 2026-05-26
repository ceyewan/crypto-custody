# 离线协作服务端

`offline-server` 是离线端的协作服务，负责用户登录、角色权限、密钥生成会话、签名会话、安全芯片登记、WebSocket 实时通信和 MPC manager 进程管理。

## 核心能力

- 用户认证：提供注册、登录、登出和令牌校验。
- 角色权限：支持管理员、协调者和参与者等离线协作角色。
- 密钥生成：创建 MPC 密钥生成会话，并协调参与方完成流程。
- 签名协作：创建签名会话，通过 WebSocket 分发和收集签名消息。
- 安全芯片管理：登记安全芯片标识和 CPLC 信息。
- 进程管理：启动并监控 `gg20_sm_manager` 后台进程。
- 本地存储：使用 SQLite 保存用户、会话、安全芯片和密钥分片相关数据。

## 目录结构

```text
offline-server/
├── bin/           # MPC manager 可执行文件
├── clog/          # 日志封装
├── docs/          # Web 和 WebSocket 模块文档
├── manager/       # 后台进程管理器
├── storage/       # SQLite 数据访问层
├── test/          # 接口和流程测试
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
WebSocket: ws://localhost:8081/ws
```

启动参数可调整端口：

```bash
go run . -web-port 8080 -ws-port 8081
```

## 本地运行

```bash
go mod tidy
go run .
```

运行前需要确认 `bin/gg20_sm_manager` 存在并具有执行权限。启动后会自动创建 `data/` 和 `logs/` 目录。

## 测试

```bash
go test ./...
```

`test/` 下的用例会访问本地服务并读写本地数据库，运行前请确认当前环境适合测试数据写入。

## 相关文档

- `docs/web_module_documentation.md`：HTTP API 模块说明。
- `docs/ws_module_documentation.md`：WebSocket 模块说明。
- `manager/README.md`：manager 进程管理器说明。
