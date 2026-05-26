# src 目录

`src` 目录保存在线 Web 前端源码。

## 目录说明

- `components/`：可复用组件。
- `router/`：页面路由和访问控制。
- `services/`：在线服务端 API 封装。
- `store/`：Vuex 全局状态。
- `views/`：业务页面。
- `App.vue`：应用根组件。
- `main.js`：应用初始化入口。

## 维护建议

- 页面级逻辑放在 `views`，跨页面复用能力优先沉淀到 `services` 或 `components`。
- 新增受保护页面时，同步检查路由权限、菜单展示和登录态处理。
