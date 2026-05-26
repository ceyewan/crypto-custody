# docs 目录

该目录保存离线协作服务端的模块说明文档。

## 文件说明

- `web_module_documentation.md`：HTTP API、用户、密钥生成、签名和安全芯片接口说明。
- `ws_module_documentation.md`：WebSocket 连接、消息类型、会话协调和实时通信说明。

## 维护建议

- 修改 HTTP 路由、请求字段或权限规则后，同步更新 Web 模块文档。
- 修改 WebSocket 消息结构、状态流转或会话协议后，同步更新 WebSocket 模块文档。
- 示例请求和消息体应尽量保持可直接用于联调。
