# services 目录

`services` 目录封装离线桌面应用前端的外部通信能力。

## 文件说明

- `api.js`：离线协作服务端 HTTP API 封装。
- `ws.js`：WebSocket 连接和实时消息处理。
- `wails-api.js`：前端调用本地 Go 能力的封装。

## 维护建议

- HTTP 接口路径变更后更新 `api.js`。
- WebSocket 消息类型变更后更新 `ws.js`，并同步检查通知和会话页面。
- 本地能力名称或参数变更后更新 `wails-api.js`。
