# 账户接口测试

该目录包含账户查询、账户创建、账户导入和管理员账户管理相关的接口测试。测试会向本地 `http://localhost:8080` 发送请求。

## 文件说明

- `common_test.go`：测试辅助函数、请求封装和管理员登录工具。
- `account_test.go`：账户查询测试。
- `create_test.go`：账户创建测试。
- `import_test.go`：账户导入测试。

## 运行前准备

1. 启动本地服务，并确认监听 `8080` 端口。
2. 设置管理员密码环境变量：

```bash
export DEFAULT_ADMIN_PASSWORD="your_admin_password"
```

## 运行方式

运行全部账户接口测试：

```bash
cd tests/account_test
go test -v
```

运行指定测试文件：

```bash
go test -v account_test.go common_test.go
go test -v create_test.go common_test.go
go test -v import_test.go common_test.go
```

运行指定测试函数：

```bash
go test -v -run TestGetAccountByAddress
```

## 覆盖范围

- 通过地址查询账户。
- 获取当前用户账户列表。
- 创建账户。
- 批量导入账户。
- 管理员查询所有账户。
- 管理员删除账户。
- 非管理员访问管理员接口的权限校验。
- 无效账户数据处理。

## 注意事项

- 测试会创建、导入和删除账户数据，请不要在生产环境运行。
- 管理员相关测试依赖默认管理员账号和正确的 `DEFAULT_ADMIN_PASSWORD`。
- 如果本地服务未启动，接口测试会失败。
