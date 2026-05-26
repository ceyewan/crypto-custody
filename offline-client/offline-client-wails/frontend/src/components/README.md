# components 目录

`components` 目录保存离线桌面应用前端的可复用组件。

## 文件说明

- `WsStatusIndicator.vue`：WebSocket 连接状态指示组件。

## 维护建议

- 组件应保持职责单一，复杂业务流程放在页面或服务封装中。
- 与连接状态相关的组件需要与 `store` 中的 WebSocket 状态保持一致。
