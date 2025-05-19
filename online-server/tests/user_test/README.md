# 用户管理模块测试

本项目包含对用户管理模块的详细客户端测试，测试会向本地启动的服务（端口8080）发送请求。

## 文件结构

- `common_test.go` - 通用测试辅助工具和函数
- `login_test.go` - 用户登录相关测试
- `register_test.go` - 用户注册相关测试
- `token_test.go` - 令牌验证相关测试
- `profile_test.go` - 用户个人资料和密码修改相关测试
- `admin_test.go` - 管理员功能相关测试

## 运行前准备

1. 确保服务端已经在本地8080端口启动
2. 设置管理员密码环境变量（用于管理员功能测试）

```bash
export DEFAULT_ADMIN_PASSWORD="your_admin_password"
```

## 运行测试

运行所有测试：

```bash
cd /Users/harrick/CodeField/crypto-custody/online-server/tests/user_test
go test -v
```

运行特定测试文件：

```bash
go test -v login_test.go common_test.go
go test -v register_test.go common_test.go
go test -v token_test.go common_test.go
go test -v profile_test.go common_test.go
go test -v admin_test.go common_test.go
```

运行特定测试函数：

```bash
go test -v -run TestLoginWithValidCredentials
```

## 测试内容概述

### 登录测试
- 测试有效凭据登录
- 测试无效用户名登录
- 测试密码错误登录
- 测试管理员登录

### 注册测试
- 测试成功注册用户
- 测试重复用户名注册
- 测试重复邮箱注册
- 测试无效注册数据

### 令牌验证测试
- 测试有效令牌验证
- 测试无效令牌验证
- 测试登出后的令牌验证

### 用户个人资料测试
- 测试获取当前用户信息
- 测试未认证访问
- 测试修改密码功能
- 测试使用错误的旧密码修改密码

### 管理员功能测试
- 测试获取所有用户
- 测试非管理员尝试获取所有用户
- 测试获取特定用户
- 测试更改用户角色
- 测试更改用户名
- 测试删除用户

## 注意事项

1. 测试会创建和修改数据库中的用户数据，请勿在生产环境中运行测试
2. 管理员功能测试需要有效的管理员账号和密码
3. 每次运行测试会生成随机用户名和邮箱，避免冲突
4. 如果服务端未运行，测试将全部失败
