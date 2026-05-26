# 离线客户端

`offline-client` 包含离线端桌面应用、安全芯片 Applet、测试 Applet 和 JavaCard 构建依赖。离线端用于保存私钥分片、访问安全芯片、参与 MPC 密钥生成和签名流程。

## 目录结构

```text
offline-client/
├── lib/                   # JavaCard SDK 和 Ant 构建扩展
├── secured/               # 带签名校验的安全芯片 Applet
├── unsecured/             # 基础数据存储 Applet 和 APDU 示例
└── offline-client-wails/  # Wails 桌面应用
```

## 模块说明

### `offline-client-wails/`

桌面应用入口，基于 Wails、Go 和 Vue 实现。应用负责用户登录、密钥生成、交易签名、通知处理、安全芯片导入和本地 MPC 调用。

### `secured/`

安全芯片 Applet，面向实际安全存储场景。读取和删除操作带签名校验，适合配合离线桌面应用使用。

### `unsecured/`

基础 Applet，用于验证 APDU 存储、读取和删除流程。该模块适合早期联调和协议验证，不应用于敏感数据存储。

### `lib/`

JavaCard 构建依赖目录，包含 `jc305u4_kit` 和 `ant-javacard.jar`，供 `secured` 和 `unsecured` 构建 Applet 使用。

## 常用入口

- `offline-client/offline-client-wails/README.md`：桌面应用使用说明。
- `offline-client/offline-client-wails/mpc_core/README.md`：MPC 核心模块说明。
- `offline-client/secured/README.md`：安全芯片 Applet 构建和部署说明。
- `offline-client/unsecured/README.md`：基础 Applet APDU 协议说明。
