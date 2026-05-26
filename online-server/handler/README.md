# handler 目录

`handler` 目录是 HTTP 请求处理层，负责参数绑定、权限上下文读取、调用 service、返回统一响应。

## 文件说明

- `user_handler.go`：登录、注册、登出、个人资料、密码修改和管理员用户管理。
- `account_handler.go`：账户查询、创建、导入、列表和删除。
- `transaction_handler.go`：余额查询、交易准备、签名发送、交易列表、详情、统计和删除。

## 编写约定

- handler 只处理请求和响应编排，复杂业务逻辑放在 `service` 或 `ethereum`。
- 返回结果优先使用 `utils` 中的统一响应工具。
- 需要当前用户信息时，从 JWT 中间件写入的上下文读取。
- 新增接口后，同步更新 `route` 和相关 API 文档。
