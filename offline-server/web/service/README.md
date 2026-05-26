# service 目录

`service` 目录承载 HTTP API 的业务逻辑，负责用户操作、密钥生成会话、签名会话和存储层调用。

## 文件说明

- `user_service.go`：用户注册、登录、角色和列表查询。
- `key_service.go`：密钥生成会话创建和参与者查询。
- `sign_service.go`：签名会话创建和地址参与者查询。

## 维护建议

- 涉及会话、用户角色和地址归属的判断应集中在 service 层。
- 数据读写通过 `storage` 层完成，避免 handler 直接访问数据库。
- 修改业务规则后同步检查 WebSocket 流程和接口测试。
