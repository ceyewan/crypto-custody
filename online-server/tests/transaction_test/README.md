# 交易管理模块测试

本项目包含对交易管理模块的详细客户端测试，测试会向本地启动的服务（端口8080）发送请求。

## 文件结构

- `common_test.go` - 通用测试辅助工具和函数
- `prepare_test.go` - 交易准备测试
- `sign_test.go` - 交易签名测试
- `send_test.go` - 交易发送测试

## 运行前准备

1. 确保服务端已经在本地8080端口启动
2. 设置管理员密码环境变量（用于管理员功能测试）

```bash
export DEFAULT_ADMIN_PASSWORD="your_admin_password"
```

3. 运行离线签名工具准备测试账户
4. 在测试账户中准备足够的ETH余额

## 运行测试

运行所有测试：

```bash
cd /Users/harrick/CodeField/crypto-custody/online-server/tests/transaction_test
go test -v
```

运行特定测试文件：

```bash
go test -v prepare_test.go common_test.go
go test -v sign_test.go common_test.go
go test -v send_test.go common_test.go
```

运行特定测试函数：

```bash
go test -v -run TestPrepareTransaction
```

## 测试内容概述

### 交易准备测试
- 测试准备交易
- 测试准备无效交易
- 测试余额不足情况

### 交易签名测试
- 测试有效签名
- 测试无效签名
- 测试签名验证

### 交易发送测试
- 测试交易发送
- 测试交易状态查询
- 测试交易确认

## 注意事项

1. 测试会创建实际的以太坊交易，请在测试网络上进行测试
2. 测试账户应当仅用于测试目的，不要在其中存放大量资金
3. 账户功能测试需要预先导入测试账户
4. 如果服务端未运行，测试将全部失败
