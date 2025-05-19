# 交易管理模块开发指南

## 1. 概述

交易管理模块提供了完整的以太坊交易功能，主要包括：

- 余额查询
- 交易准备
- 交易签名
- 交易发送
- 交易状态监控

本文档主要面向开发人员，提供模块的设计思路、架构说明和开发指引。

## 2. 模块结构

交易管理模块采用分层架构设计，包含以下几个主要层次：

- **以太坊交互层 (ethereum)**: 与区块链直接交互
- **服务层 (service)**: 实现业务逻辑
- **处理器层 (handler)**: 处理HTTP请求和响应
- **路由层 (route)**: 定义API路由和权限控制
- **数据层 (model)**: 定义交易相关的数据模型

### 2.1 文件结构

```
online-server/
├── ethereum/
│   ├── client.go           # 以太坊客户端
│   └── transaction_manager.go # 交易管理器
├── service/
│   └── transaction_service.go # 交易服务
├── handler/
│   └── transaction_handler.go # 交易请求处理
├── route/
│   └── transaction_router.go  # 交易路由定义
├── model/
│   └── transaction_model.go   # 交易模型定义
└── dto/
    └── transaction_dto.go     # 交易数据传输对象
```

## 3. 交易流程

### 3.1 准备交易流程

1. 客户端提交发送方地址、接收方地址和金额信息
2. 系统检查发送方是否有正在处理的交易
3. 系统创建交易对象，包括获取nonce、估算gas等
4. 系统生成消息哈希并返回给客户端
5. 交易状态标记为 "Created"

### 3.2 签名与发送流程

1. 客户端提交消息哈希和签名数据
2. 系统验证签名是否有效
3. 系统将已签名的交易发送到以太坊网络
4. 交易状态标记为 "Submitted"
5. 系统启动一个后台任务监控交易确认状态
6. 交易确认后，状态更新为 "Confirmed" 或 "Failed"

## 4. 核心组件介绍

### 4.1 Client

`ethereum/client.go` 封装了与以太坊网络的基础交互功能：

- 连接以太坊节点
- 获取账户余额
- 获取账户nonce
- 估算gas价格
- 发送原始交易
- 查询交易状态

### 4.2 TransactionManager

`ethereum/transaction_manager.go` 管理交易的完整生命周期：

- 创建和准备交易
- 验证签名
- 发送交易
- 监控交易状态
- 管理交易缓存

### 4.3 TransactionService

`service/transaction_service.go` 负责交易数据的持久化存储：

- 创建交易记录
- 更新交易状态
- 查询交易历史
- 查询交易详情

## 5. 开发指南

### 5.1 添加新的交易类型

如需添加新的交易类型，请按照以下步骤：

1. 在 `model/transaction_model.go` 中扩展交易模型
2. 在 `ethereum/transaction_manager.go` 中添加新的交易处理方法
3. 在 `service/transaction_service.go` 中添加相应的服务方法
4. 在 `handler/transaction_handler.go` 中实现处理函数
5. 在 `route/transaction_router.go` 中添加新的API路由

### 5.2 处理交易错误

交易过程中的错误处理非常重要，主要错误类型包括：

1. **请求参数错误**: 地址格式不正确、金额无效等
2. **业务逻辑错误**: 余额不足、nonce冲突等
3. **网络错误**: 连接以太坊节点失败
4. **交易错误**: 交易执行失败、gas不足等

对于每种错误类型，应该提供清晰的错误信息和恰当的HTTP状态码。

### 5.3 监控与维护

为确保系统稳定运行，应该定期执行以下维护任务：

1. 检查所有待处理交易的状态
2. 清理已完成交易的缓存
3. 监控以太坊网络连接状态
4. 检查过期交易并处理

建议通过定时任务来执行这些维护操作。

## 6. 安全最佳实践

开发交易功能时，请遵循以下安全最佳实践：

1. **离线签名**: 私钥应在离线环境中保存和使用
2. **交易确认**: 等待足够的区块确认再认为交易成功
3. **金额校验**: 对交易金额进行严格的校验和限制
4. **防重放攻击**: 确保交易nonce正确使用
5. **日志记录**: 详细记录所有交易操作
6. **错误处理**: 避免在错误信息中泄露敏感信息

## 7. 测试指南

交易管理模块有两种测试方式：

### 7.1 单元测试

执行单元测试：

```bash
cd online-server
go test -v ./tests/transaction_test.go
```

### 7.2 集成测试

执行集成测试：

```bash
cd online-server
go test -v ./tests/transaction_integration_test.go
```

## 8. 常见问题与排错

1. **交易未确认**: 检查gas价格是否过低或网络是否拥堵
2. **nonce错误**: 检查账户的当前nonce值是否正确
3. **签名验证失败**: 确认签名数据格式和对应的消息哈希是否匹配
4. **余额不足**: 确认账户余额是否足够支付交易金额和gas费用

## 9. API 参考

请参考 [交易管理API文档](transaction_management_api.md) 获取完整的API说明。
