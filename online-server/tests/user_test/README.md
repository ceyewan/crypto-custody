# 用户接口测试

该目录包含用户注册、登录、令牌验证、个人资料、密码修改和管理员用户管理相关的接口测试。测试会向本地 `http://localhost:8080` 发送请求。

## 文件说明

- `common_test.go`：测试辅助函数、请求封装和管理员登录工具。
- `login_test.go`：登录相关测试。
- `register_test.go`：注册相关测试。
- `token_test.go`：令牌校验相关测试。
- `profile_test.go`：个人资料和密码修改测试。
- `admin_test.go`：管理员功能测试。

## 运行前准备

1. 启动本地服务，并确认监听 `8080` 端口。
2. 设置管理员密码环境变量：

```bash
export DEFAULT_ADMIN_PASSWORD="your_admin_password"
```

## 运行方式

运行全部用户接口测试：

```bash
cd tests/user_test
go test -v
```

运行指定测试文件：

```bash
go test -v login_test.go common_test.go
go test -v register_test.go common_test.go
go test -v token_test.go common_test.go
go test -v profile_test.go common_test.go
go test -v admin_test.go common_test.go
```

运行指定测试函数：

```bash
go test -v -run TestLoginWithValidCredentials
```

## 覆盖范围

- 正确和错误凭据登录。
- 用户注册、重复用户名、重复邮箱和无效注册数据。
- 有效令牌、无效令牌和登出后的令牌验证。
- 当前用户信息查询和密码修改。
- 管理员查看用户、修改角色、修改用户名、修改密码和删除用户。

## 注意事项

- 测试会创建和修改用户数据，请不要在生产环境运行。
- 管理员相关测试依赖默认管理员账号和正确的 `DEFAULT_ADMIN_PASSWORD`。
- 测试会生成随机用户名和邮箱，用于避免重复数据冲突。
