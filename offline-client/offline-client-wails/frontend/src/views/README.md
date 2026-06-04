# views 目录

`views` 目录保存离线桌面应用的业务页面。

## 页面说明

- `Login.vue`：登录。
- `Register.vue`：注册。
- `Dashboard.vue`：仪表板。
- `Users.vue`：用户管理。
- `OfflineTasks.vue`：离线任务导入、发起和结果导出。
- `Notifications.vue`：通知消息。
- `ImportSE.vue`：安全芯片导入。

## 维护建议

- 页面内的接口调用优先通过 `services` 完成。
- 涉及本地文件或安全芯片的能力应通过 Wails 封装调用。
- 新增流程页面后同步检查通知、实时消息和权限。
