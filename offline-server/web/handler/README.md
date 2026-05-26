# handler 目录

`handler` 目录是离线协作服务端的 HTTP 请求处理层，负责解析请求、调用 service、返回统一 JSON 响应。

## 文件说明

- `handler.go`：通用响应结构和辅助方法。
- `user_handler.go`：登录、注册、登出、令牌校验和用户管理。
- `key_handler.go`：密钥生成会话创建和可用参与者查询。
- `sign_handler.go`：签名会话创建和地址参与者查询。
- `se_handler.go`：安全芯片登记。

## 维护建议

- 参数校验失败时应返回清晰错误信息。
- 需要当前用户和角色时，从认证中间件写入的上下文读取。
- 复杂状态流转不要写在 handler 中，放入 `web/service` 或 `storage`。
