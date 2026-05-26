# service 目录

`service` 目录承载主要业务逻辑，负责数据库读写、状态流转和跨模块协调。

## 文件说明

- `user_service.go`：用户注册、登录、登出、密码修改、角色和管理员操作。
- `account_service.go`：账户创建、导入、查询和删除。
- `transaction_service.go`：交易记录创建、状态更新、列表查询和统计。

## 维护建议

- 业务规则优先放在 service，handler 只做请求和响应编排。
- 数据库操作应通过 `utils.GetDB()` 获取连接。
- 涉及交易状态、账户归属或权限边界的改动，需要补充对应接口测试。
