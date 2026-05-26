# web 目录

`web` 目录实现离线协作服务端的 HTTP API，负责用户认证、密钥生成会话创建、签名会话创建、安全芯片登记和权限中间件。

## 文件说明

- `web.go`：HTTP 服务启动和优雅关闭。
- `router.go`：Gin 路由注册、认证中间件、角色权限和 CORS 配置。
- `handler/`：HTTP 请求处理入口。
- `service/`：用户、密钥、签名等业务逻辑。

## 主要路由

- `/user/login`：用户登录。
- `/user/register`：用户注册。
- `/user/checkAuth`：校验登录态。
- `/user/admin/users`：管理员用户列表和角色管理。
- `/keygen/create/:initiator`：创建密钥生成会话。
- `/keygen/users`：获取可参与密钥生成的用户。
- `/sign/create/:initiator`：创建签名会话。
- `/sign/users/:address`：获取指定地址可参与签名的用户。
- `/se/create`：登记安全芯片信息。

## 维护建议

- 路由权限应与用户角色定义保持一致。
- handler 中只做请求解析和响应组织，业务规则放入 `service`。
- 接口变更后同步更新 `docs/web_module_documentation.md`。
