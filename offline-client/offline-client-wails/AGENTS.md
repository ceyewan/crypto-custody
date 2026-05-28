# AGENTS.md

本目录是离线系统桌面端。它是 Wails 应用，包含 Vue 前端、Go 绑定、本地 MPC core、SE client，以及内嵌的 `gg20_keygen` / `gg20_signing` 跨平台二进制。

## 开发原则

- 桌面端必须允许未插入 SE 时正常打开；只有读取 CPLC、keygen、sign、destroy 时才提示或返回 SE 错误。
- 服务器地址、WebSocket 地址、登录信息和读卡器名称由客户端设置驱动，不能写死为公网或本机固定地址。
- MPC 本地临时文件必须写入用户可写目录或调用方显式传入的临时目录，不要使用打包后可能只读的相对路径。
- 本地分片文件只作为临时文件使用，执行完成后清理；持久化的是服务端保存的加密 shard 和 SE 中按 `record_id` 保存的随机密钥。
- `record_id` 是 SE 内部记录编号，也是同一张 SE 上隔离不同参与方/密钥材料的关键字段，不要退回 username 作为 SE 存储主键。
- 本地单 SE 跑三参与方是测试便利能力；不要引入跨进程 SE 文件锁，除非用户明确重新要求。

## 常用命令

后端和 MPC core 测试：

```bash
go test ./...
```

前端构建：

```bash
cd frontend
npm run build
```

macOS 本地打包：

```bash
wails build -clean -platform darwin/arm64
```

打包后如需本机模拟多参与方，可以复制 `.app` 并修改 `CFBundleIdentifier`，确保 u1/u2/u3 的 localStorage 互相隔离。

## 目录说明

- `frontend/src/services/settings.js`: HTTP/WS/读卡器配置。
- `frontend/src/services/ws.js`: WebSocket 消息、邀请确认、任务状态和结果回传。
- `wails_services.go`: 前端调用 Go 本地能力的主要入口。
- `mpc_core/services/`: MPC 和 SE 编排逻辑。
- `mpc_core/seclient/`: SE APDU 客户端。
- `mpc_core/utils/binaries/`: 内嵌 GG20 二进制来源，更新时必须确认 macOS、Linux、Windows 三端匹配同一 release。

## 验证重点

- 修改 Wails 绑定或 Go models 后，重新生成/检查 `frontend/wailsjs`。
- 修改前端连接或消息字段后，确认 HTTP 地址和 WebSocket 地址都能从客户端设置生效。
- 修改 MPC/SE 流程后，至少跑 `go test ./...`；涉及 UI 时再跑 `frontend/npm run build` 和 Wails 打包。
- 实机测试时先确认服务端 `8080/8081` 都在监听，WebSocket 地址应是 `ws://127.0.0.1:8081/ws`，不是 HTTP 端口 `8080`。
