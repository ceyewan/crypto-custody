# 以太坊交易管理模块 (ethereum)

![版本](https://img.shields.io/badge/版本-1.0.0-blue)
![go-ethereum](https://img.shields.io/badge/go--ethereum-1.15.11-orange)

## 简介

`ethereum` 包提供与以太坊区块链的完整交互功能，实现了交易准备、签名和发送的全流程。该模块采用在线-离线分离的安全架构，显著提高了私钥管理的安全性，适用于加密资产托管系统。

本模块是整个交易管理系统的核心部分，负责处理与区块链的所有直接交互，包括连接节点、构建交易、验证签名、发送交易以及监控交易状态等。

## 主要特性

- 安全的交易处理流程，支持在线-离线分离签名
- 完整的交易生命周期管理（创建、签名、发送、确认）
- 自动的交易状态监控和更新
- 防止重复交易的保护机制
- 支持交易历史查询和管理

## 架构设计

该包主要由两个核心组件组成：

1. **Client**: 封装以太坊网络连接和基础操作
2. **TransactionManager**: 管理交易的完整生命周期

### 交易流程

```
创建交易 -> 生成消息哈希 -> 离线签名 -> 提交签名 -> 发送交易 -> 监控确认
```

## 使用指南

### 初始化客户端

```go
// 使用默认配置（Sepolia测试网）
client, err := ethereum.GetClientInstance()
if err != nil {
    panic(err)
}

// 或使用自定义配置
config := ethereum.ClientConfig{
    RPC:         "https://mainnet.infura.io/v3/YOUR_API_KEY",
    ChainID:     big.NewInt(1), // 以太坊主网
    ConfirmTime: 120 * time.Second,
}

client, err = ethereum.GetClientInstance(config)
if err != nil {
    panic(err)
}
```

### 创建交易管理器

```go
txManager := ethereum.NewTransactionManager(client)
```

### 创建交易

```go
fromAddress := "0x123...456" // 发送方地址
toAddress := "0x789...abc"   // 接收方地址
amount := new(big.Float).SetFloat64(0.01) // 转账金额（ETH）

txID, messageHash, err := txManager.CreateTransaction(fromAddress, toAddress, amount)
if err != nil {
    log.Fatalf("创建交易失败: %v", err)
}

fmt.Printf("交易已创建，ID: %d, 消息哈希: %s\n", txID, messageHash)
```

### 处理签名

```go
// 消息哈希需要离线设备进行签名
// 获取到签名后进行验证和处理
signature := "1a2b3c..." // 从离线设备获取的签名

txID, err := txManager.SignTransaction(messageHash, signature)
if err != nil {
    log.Fatalf("处理签名失败: %v", err)
}

fmt.Printf("交易签名成功，ID: %d\n", txID)
```

### 发送交易

```go
txID, txHash, err := txManager.SendTransaction(messageHash)
if err != nil {
    log.Fatalf("发送交易失败: %v", err)
}

fmt.Printf("交易已发送，ID: %d, 交易哈希: %s\n", txID, txHash)
```

### 查询交易状态

```go
status, err := txManager.GetTransactionStatus(messageHash)
if err != nil {
    log.Fatalf("查询状态失败: %v", err)
}

fmt.Printf("交易状态: %s\n", status.String())
```

### 定期维护

```go
// 检查待处理交易状态，建议通过定时任务定期执行
if err := txManager.CheckPendingTransactions(); err != nil {
    log.Printf("检查待处理交易失败: %v", err)
}

// 清理已完成的交易缓存，释放内存
txManager.ClearCompletedTransactions()
```

## 错误处理

该模块定义了多个特定错误类型，便于进行错误处理：

- `ErrTransactionNotFound`: 交易记录不存在
- `ErrTransactionAlreadySent`: 交易已经发送，无法重复处理
- `ErrInvalidSignature`: 提供的签名无效或与发送者不匹配
- `ErrTransactionInProgress`: 用户已有正在处理中的交易

示例错误处理：

```go
_, _, err := txManager.CreateTransaction(fromAddress, toAddress, amount)
if err != nil {
    if errors.Is(err, ethereum.ErrTransactionInProgress) {
        fmt.Println("用户有正在处理的交易，请等待完成")
        return
    }
    log.Fatalf("创建交易失败: %v", err)
}
```

## 最佳实践

1. **交易生命周期管理**：使用 `GetTransactionStatus` 定期检查交易状态
2. **定期维护**：通过定时任务调用 `CheckPendingTransactions` 和 `ClearCompletedTransactions`
3. **错误处理**：详细处理各种交易错误情况，特别是网络错误和签名错误
4. **安全建议**：私钥管理应在离线环境中进行，只将签名结果传递到在线环境

## 完整示例

```go
package main

import (
    "fmt"
    "log"
    "math/big"
    
    "online-server/ethereum"
)

func main() {
    // 初始化客户端
    client, err := ethereum.GetClientInstance()
    if err != nil {
        log.Fatalf("初始化以太坊客户端失败: %v", err)
    }
    defer client.Close()
    
    // 创建交易管理器
    txManager := ethereum.NewTransactionManager(client)
    
    // 创建交易
    fromAddress := "0x123...456"
    toAddress := "0x789...abc"
    amount := new(big.Float).SetFloat64(0.01) // 0.01 ETH
    
    txID, messageHash, err := txManager.CreateTransaction(fromAddress, toAddress, amount)
    if err != nil {
        log.Fatalf("创建交易失败: %v", err)
    }
    
    fmt.Printf("交易已创建，ID: %d, 消息哈希: %s\n", txID, messageHash)
    
    // 假设已获取签名
    signature := "..." // 从离线设备获取的签名
    
    // 处理签名
    txID, err = txManager.SignTransaction(messageHash, signature)
    if err != nil {
        log.Fatalf("处理签名失败: %v", err)
    }
    
    // 发送交易
    txID, txHash, err := txManager.SendTransaction(messageHash)
    if err != nil {
        log.Fatalf("发送交易失败: %v", err)
    }
    
    fmt.Printf("交易已发送，ID: %d, 交易哈希: %s\n", txID, txHash)
}
```

## 注意事项

1. 该模块默认连接到 Sepolia 测试网，生产环境需更改配置
2. 确保网络连接稳定，以便正确监控交易状态
3. 交易签名应在安全的离线环境中进行
4. 大额交易建议先进行小额测试

## 依赖项

- github.com/ethereum/go-ethereum: v1.15.11
- 内部依赖: 
  - online-server/model
  - online-server/service
  - online-server/utils

## 贡献

如需对本模块贡献代码，请确保理解以太坊交易的完整生命周期，并确保所有更改都经过充分测试。由于本模块直接处理数字资产交易，安全性是首要考虑因素。

## 版本历史

- v1.0.0 (2025-05-19): 初始版本
  - 实现基本交易功能
  - 支持在线-离线分离架构
  - 添加交易状态监控

## 主要贡献者

- [Your Name] - 模块设计与实现