# Crypto Custody

Crypto Custody 是一套加密资产托管系统，采用在线与离线分离的架构。在线端负责账户、用户、交易构建和链上广播；离线端负责多方计算、签名协作和安全芯片存储，私钥材料不在在线服务中保存。

## 系统组成

```text
crypto-custody/
├── online-server/      # 在线服务端，提供用户、账户、交易和以太坊交互接口
├── online-client/      # 在线 Web 前端，面向账户和交易管理
├── offline-server/     # 离线协作服务端，协调 MPC 密钥生成和签名会话
└── offline-client/     # 离线客户端与安全芯片 Applet
```

## 工作流程

1. 在线端创建用户、管理账户并准备待签名交易。
2. 在线端生成交易哈希或待签名消息，不接触私钥。
3. 离线端通过 MPC 流程生成密钥分片，并在需要时协同完成签名。
4. 签名结果提交回在线端，由在线端验证签名地址并广播交易。
5. 在线端持续查询交易状态，并维护交易记录。

## 目录说明

### `online-server/`

在线服务端基于 Go 实现，提供认证、角色权限、账户管理、交易管理、Sepolia 交易广播和状态查询。详细说明见 `online-server/README.md`。

### `online-client/`

在线 Web 前端基于 Vue 2 和 Element UI 实现，提供登录、用户管理、账户管理、交易管理和个人资料等页面。详细说明见 `online-client/README.md`。

### `offline-server/`

离线协作服务端基于 Go 实现，提供 HTTP API、WebSocket 协调服务、MPC manager 进程管理和本地数据持久化。详细说明见 `offline-server/README.md`。

### `offline-client/`

离线客户端包含 Wails 桌面应用、安全芯片 Applet、测试 Applet 和 JavaCard 构建依赖。详细说明见 `offline-client/README.md`。

## 运行顺序建议

本地联调时建议按以下顺序启动：

1. 启动 `online-server`，确认在线 API 可访问。
2. 启动 `offline-server`，确认 HTTP 与 WebSocket 服务可访问。
3. 启动 `online-client/frontend`，进行账户和交易管理。
4. 启动 `offline-client/offline-client-wails`，进行 MPC 密钥生成和签名协作。

## 文档入口

- `online-server/API_DOCUMENTATION.md`：在线服务端汇总接口文档。
- `online-server/docs/`：在线服务端分模块接口与开发说明。
- `offline-server/docs/`：离线协作服务端 Web 和 WebSocket 模块说明。
- `offline-client/offline-client-wails/DEVELOPMENT.md`：离线桌面应用开发说明。
- `offline-client/offline-client-wails/mpc_core/DEVELOPMENT.md`：MPC 核心模块开发说明。
- `offline-client/secured/DEVELOPMENT.md`：安全芯片 Applet 开发说明。
