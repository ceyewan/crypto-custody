# src 目录

`src` 目录保存离线桌面应用前端源码，负责登录、通知、密钥生成、签名、安全芯片导入和测试页面。

## 目录说明

- `components/`：可复用组件。
- `router/`：页面路由。
- `services/`：HTTP、WebSocket 和 Wails API 封装。
- `store/`：Vuex 全局状态。
- `views/`：业务页面。
- `App.vue`：应用根组件。
- `main.js`：应用初始化入口。

## 维护建议

- 与本地 Go 能力交互时优先通过 `services/wails-api.js`。
- 与离线协作服务通信时优先通过 `services/api.js` 和 `services/ws.js`。
- 页面权限和角色变化需要同步检查路由、状态和菜单展示。
