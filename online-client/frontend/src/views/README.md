# views 目录

`views` 目录保存在线前端的业务页面。

## 页面说明

- `Login.vue`：登录。
- `Register.vue`：注册。
- `Dashboard.vue`：仪表板。
- `Users.vue`：管理员用户管理。
- `Accounts.vue`：账户管理。
- `Transactions.vue`：交易管理。
- `Profile.vue`：个人资料。

## 维护建议

- 页面中只保留展示和交互编排，接口调用优先通过 `services` 完成。
- 新增页面后，同步更新路由、权限和必要的导航入口。
