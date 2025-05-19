# 账户管理模块测试

本项目包含对账户管理模块的详细客户端测试，测试会向本地启动的服务（端口8080）发送请求。

## 文件结构

- `common_test.go` - 通用测试辅助工具和函数
- `account_test.go` - 账户查询测试
- `create_account_test.go` - 账户创建测试
- `import_account_test.go` - 账户导入测试

## 运行前准备

1. 确保服务端已经在本地8080端口启动
2. 设置管理员密码环境变量（用于管理员功能测试）

```bash
export DEFAULT_ADMIN_PASSWORD="your_admin_password"
```

## 运行测试

运行所有测试：

```bash
cd /Users/harrick/CodeField/crypto-custody/online-server/tests/account_test
go test -v
```

运行特定测试文件：

```bash
go test -v account_test.go common_test.go
go test -v create_account_test.go common_test.go
go test -v import_account_test.go common_test.go
```

运行特定测试函数：

```bash
go test -v -run TestGetAccountByAddress
```

## 测试内容概述

### 账户查询测试
- 测试通过地址查询账户
- 测试获取用户账户列表
- 测试非管理员获取所有账户
- 测试获取不存在的账户

### 账户创建测试
- 测试创建新账户
- 测试普通用户创建账户
- 测试创建无效账户

### 账户导入测试
- 测试导入单个账户
- 测试批量导入账户
- 测试导入无效账户
- 测试普通用户导入账户

## 注意事项

1. 测试会创建和修改数据库中的账户数据，请勿在生产环境中运行测试
2. 管理员功能测试需要有效的管理员账号和密码
3. 如果服务端未运行，测试将全部失败