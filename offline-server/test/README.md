# test 目录

`test` 目录保存离线协作服务端的接口和流程测试。

## 文件说明

- `user_test.go`：用户注册、登录和权限相关测试。
- `keygen_test.go`：密钥生成会话相关测试。
- `sign_test.go`：签名会话相关测试。
- `se_test.go`：安全芯片登记相关测试。

## 运行方式

```bash
go test ./test -v
```

也可以在服务端目录运行全部测试：

```bash
go test ./...
```

## 注意事项

- 测试会读写本地数据库。
- 涉及 WebSocket 和 MPC manager 的测试需要先确认相关服务可用。
