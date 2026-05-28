# AGENTS.md

本目录是离线系统服务端。这里负责 HTTP API、WebSocket 协调、离线任务包导入/导出、MPC 会话编排、manager 进程生命周期、数据库和审计记录。

## 开发原则

- 以测试大纲要求的存管提控能力为边界，避免提前引入复杂平台化设计。
- 服务端只做协调、授权、状态和密文数据持久化，不接触 SE 内部明文密钥。
- 每次 keygen/sign 会话应启动独立 `gg20_sm_manager`，会话结束或失败后释放进程和端口。
- `manager_addr` 必须使用客户端可访问的地址，不能把服务端内部绑定地址误下发给桌面端。
- 本地单 SE 多参与方只是联调便利能力，依赖不同 `record_id` 隔离记录；不要假设真实生产会共用一张 SE。
- 不要引入跨进程 SE 文件锁，除非用户明确重新要求。

## 常用命令

```bash
go test ./...
go build -o bin/offline-server .
```

本地开发常用启动方式：

```bash
OFFLINE_MANAGER_PUBLIC_HOST=127.0.0.1 \
OFFLINE_MANAGER_BIND_ADDRESS=0.0.0.0 \
OFFLINE_MANAGER_PORT_START=18001 \
OFFLINE_MANAGER_PORT_END=18100 \
./bin/offline-server -web-port 8080 -ws-host 0.0.0.0 -ws-port 8081
```

默认本机地址：

- HTTP: `http://127.0.0.1:8080`
- WebSocket: `ws://127.0.0.1:8081/ws`

## 数据和安全

- SQLite 数据库默认在 `data/crypto-custody.db`。
- manager 日志在 `logs/managers/`，服务端开发日志通常在 `logs/offline-server-dev.log`。
- 私钥、测试密钥、数据库和日志通常是本地运行产物，提交前确认不要误提交敏感数据或大体积运行产物。
- 在线/离线系统交换的数据包格式以 `docs/` 下的契约和设计文档为准，改字段时同步更新文档和桌面端处理逻辑。

## 验证重点

- 登录、CORS、HTTP API 和 WebSocket 分别验证，不要只看一个端口。
- keygen/sign 邀请流程要覆盖接受、拒绝、错误回传、断线重连和重复消息。
- 修改 manager 或 MPC 编排后，至少跑服务端 `go test ./...`，并用桌面端或 smoke 工具走一次 2-of-3 keygen/sign。
