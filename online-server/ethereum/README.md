# Ethereum 模块

`ethereum` 包封装在线端与以太坊网络的交互逻辑，包括节点连接、余额查询、交易构建、签名验证、广播交易和交易确认检查。

## 文件说明

- `client.go`：以太坊客户端封装，负责连接节点、查询余额、获取 nonce、估算 gas、广播交易和查询收据。
- `transaction_manager.go`：交易生命周期管理，负责创建待签名交易、处理签名、发送交易和检查交易状态。

## 默认网络配置

默认配置连接 Sepolia 测试网：

```go
ClientConfig{
    RPC:         "https://sepolia.infura.io/v3/" + os.Getenv("ETH_RPC"),
    ChainID:     big.NewInt(11155111),
    ConfirmTime: 60 * time.Second,
}
```

运行前需要设置 `ETH_RPC`。该变量应为 Infura 项目标识，不需要包含完整 URL。

## 交易生命周期

```text
创建交易 -> 生成消息哈希 -> 离线签名 -> 提交签名 -> 广播交易 -> 查询确认状态
```

在线端不保存私钥。签名应由离线环境生成，在线端只接收签名结果并验证签名是否匹配发送方地址。

## 常用调用

### 获取客户端

```go
client, err := ethereum.GetClientInstance()
if err != nil {
    return err
}
defer client.Close()
```

### 查询余额

```go
balance, err := client.GetBalance("0x...")
if err != nil {
    return err
}
```

### 创建并发送交易

```go
txManager := ethereum.NewTransactionManager(client)

txID, messageHash, err := txManager.CreateTransaction(fromAddress, toAddress, amount)
if err != nil {
    return err
}

txID, err = txManager.SignTransaction(messageHash, signature)
if err != nil {
    return err
}

txID, txHash, err := txManager.SendTransaction(messageHash)
if err != nil {
    return err
}
```

## 维护任务

- 定期调用 `CheckPendingTransactions` 检查已广播交易的确认状态。
- 定期调用 `ClearCompletedTransactions` 清理已完成交易的内存缓存。
- 对网络错误、签名错误、重复发送等场景进行明确处理。

## 常见错误

- `ErrTransactionNotFound`：交易记录不存在。
- `ErrTransactionAlreadySent`：交易已经发送，不能重复发送。
- `ErrInvalidSignature`：签名无效，或签名地址与发送方地址不匹配。
- `ErrTransactionInProgress`：同一发送方已有未完成交易。

## 注意事项

- 生产环境切换网络前，需要同时确认 RPC 地址、链 ID 和交易确认策略。
- 交易测试和调试必须使用测试网络资产。
- 大额或高风险操作应先使用小额交易验证完整流程。
