# docs 目录

该目录保存离线协作服务端的模块说明文档。

## 文件说明

- `web_module_documentation.md`：HTTP API、用户、密钥生成、签名和安全芯片接口说明。
- `ws_module_documentation.md`：WebSocket 连接、消息类型、会话协调和实时通信说明。
- `online_offline_mpc_contract.md`：在线/离线任务包和结果包格式、数据边界、离线服务端数据库、桌面端与 SE 的 keygen/sign 流程。
- `offline_system_detailed_design.md`：离线系统详细设计，覆盖测试大纲所需的 MPC、SE、桌面端、服务端、认证、移交、销毁和提取控制边界。
- `offline_server_design.md`：离线服务端目标设计、GG20 会话级 manager、任务包、重试和当前实现差距。
- `zengo_multi_party_ecdsa_usage.md`：ZenGo-X GG20 工具的编译、keygen、signing 命令和本项目封装要求。

## 维护建议

- 修改 HTTP 路由、请求字段或权限规则后，同步更新 Web 模块文档。
- 修改 WebSocket 消息结构、状态流转或会话协议后，同步更新 WebSocket 模块文档。
- 示例请求和消息体应尽量保持可直接用于联调。
