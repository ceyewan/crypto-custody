# storage 目录

`storage` 目录是离线协作服务端的数据访问层，负责用户、会话、安全芯片和密钥分片等数据的持久化。

## 文件说明

- `interfaces.go`：存储接口定义。
- `errors.go`：存储层错误定义。
- `user_storage.go`：用户数据读写。
- `case_storage.go`：业务案例或流程数据读写。
- `keygen_storage.go`：密钥生成会话数据读写。
- `sign_storage.go`：签名会话数据读写。
- `share_storage.go`：密钥分片数据读写。
- `se_storage.go`：安全芯片数据读写。
- `db/`：数据库连接初始化。
- `model/`：数据库模型定义。

## 维护建议

- 存储层应返回可判断的错误，便于 service 层转换为业务响应。
- 数据模型字段变更后，同步检查初始化、迁移和测试数据。
- 私钥分片和敏感字段需要保持加密或受控存储。
