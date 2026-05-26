# route 目录

`route` 目录集中注册 HTTP 路由，并为不同接口配置认证和角色权限。

## 文件说明

- `router.go`：统一入口，挂载全局中间件并注册各业务路由。
- `user_router.go`：用户相关路由。
- `account_router.go`：账户相关路由。
- `transaction_router.go`：交易相关路由。

## 路由分组

- `/api`：登录、注册、令牌检查等公开接口。
- `/api/users`：用户资料和管理员用户管理。
- `/api/accounts`：账户查询、创建、导入和管理员账户管理。
- `/api/transaction`：余额、交易准备、签名发送、交易查询和统计。

## 维护建议

- 新接口应放入语义清晰的路由组，并明确是否需要 JWT、管理员或警员权限。
- 路由路径变更后，同步更新 handler 测试和 API 文档。
