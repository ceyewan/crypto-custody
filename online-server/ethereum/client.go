// Package ethereum 提供与以太坊区块链交互的功能，实现了交易准备、签名和发送的完整流程。
// 该包支持在线-离线分离的交易模式，提高了私钥管理的安全性。
package ethereum

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	// clientInstance 存储Client的单例实例
	clientInstance     *Client
	// clientInstanceOnce 确保Client只被初始化一次
	clientInstanceOnce sync.Once
)

// ClientConfig 定义以太坊客户端的配置参数
type ClientConfig struct {
	// RPC 以太坊节点的RPC地址
	RPC         string
	// ChainID 区块链网络ID
	ChainID     *big.Int
	// ConfirmTime 等待交易确认的超时时间
	ConfirmTime time.Duration
}

// DefaultConfig 返回预设的默认配置，使用Sepolia测试网
// 
// 返回:
//   - ClientConfig: 包含默认参数的配置对象
func DefaultConfig() ClientConfig {
	return ClientConfig{
		RPC:         "https://sepolia.infura.io/v3/766c230ed91a48a097e2739b966bbbf7",
		ChainID:     big.NewInt(11155111), // Sepolia 测试网
		ConfirmTime: 60 * time.Second,     // 等待交易确认的默认时间
	}
}

// Client 封装了以太坊客户端的功能，提供与区块链交互的方法
type Client struct {
	// client 底层的以太坊客户端
	client  *ethclient.Client
	// config 客户端配置
	config  ClientConfig
	// chainID 当前连接的网络ID
	chainID *big.Int
	// mu 用于保护并发访问的互斥锁
	mu      sync.Mutex
}

// GetClientInstance 获取Client的单例实例，确保全局只有一个客户端连接
// 
// 返回:
//   - *Client: 客户端实例
//   - error: 初始化过程中的错误
func GetClientInstance() (*Client, error) {
	var err error
	clientInstanceOnce.Do(func() {
		clientInstance, err = newClient(DefaultConfig())
	})

	if err != nil {
		return nil, err
	}

	return clientInstance, nil
}

// newClient 创建一个新的以太坊客户端实例并进行初始化
// 
// 参数:
//   - config: 客户端配置参数
//
// 返回:
//   - *Client: 初始化成功的客户端实例
//   - error: 初始化过程中的错误
func newClient(config ClientConfig) (*Client, error) {
	client, err := ethclient.Dial(config.RPC)
	if err != nil {
		return nil, fmt.Errorf("连接以太坊节点失败: %w", err)
	}

	// 如果未指定 chainID，则从网络获取
	var chainID *big.Int
	if config.ChainID != nil {
		chainID = config.ChainID
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		chainID, err = client.NetworkID(ctx)
		if err != nil {
			return nil, fmt.Errorf("获取链ID失败: %w", err)
		}
	}

	return &Client{
		client:  client,
		config:  config,
		chainID: chainID,
	}, nil
}

// GetBalance 获取指定地址的ETH余额
// 
// 参数:
//   - address: 以太坊地址（十六进制字符串）
//
// 返回:
//   - *big.Float: 以ETH为单位的余额
//   - error: 查询过程中的错误
func (c *Client) GetBalance(address string) (*big.Float, error) {
	addr := common.HexToAddress(address)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	balance, err := c.client.BalanceAt(ctx, addr, nil)
	if err != nil {
		return nil, fmt.Errorf("获取余额失败: %w", err)
	}

	// 将Wei转换为ETH (1 ETH = 10^18 Wei)
	fbalance := new(big.Float)
	fbalance.SetString(balance.String())
	ethValue := new(big.Float).Quo(fbalance, big.NewFloat(1e18))
	return ethValue, nil
}

// GetNonce 获取指定账户的下一个可用nonce值
// 
// 参数:
//   - address: 以太坊地址（十六进制字符串）
//
// 返回:
//   - uint64: 账户的nonce值
//   - error: 查询过程中的错误
func (c *Client) GetNonce(address string) (uint64, error) {
	addr := common.HexToAddress(address)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return c.client.PendingNonceAt(ctx, addr)
}

// SuggestGasPrice 获取网络推荐的gas价格
// 
// 返回:
//   - *big.Int: 推荐的gas价格（单位：Wei）
//   - error: 查询过程中的错误
func (c *Client) SuggestGasPrice() (*big.Int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return c.client.SuggestGasPrice(ctx)
}

// SendTransaction 将签名后的交易广播到以太坊网络
// 
// 参数:
//   - tx: 已签名的交易对象
//
// 返回:
//   - error: 交易发送过程中的错误
func (c *Client) SendTransaction(tx *types.Transaction) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return c.client.SendTransaction(ctx, tx)
}

// WaitForConfirmation 等待交易被网络确认并返回交易收据
// 
// 参数:
//   - tx: 要等待确认的交易对象
//
// 返回:
//   - *types.Receipt: 交易收据，包含执行结果和gas消耗等信息
//   - error: 等待过程中的错误或超时
func (c *Client) WaitForConfirmation(tx *types.Transaction) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.ConfirmTime)
	defer cancel()

	return bind.WaitMined(ctx, c.client, tx)
}

// Close 关闭客户端连接并释放相关资源
func (c *Client) Close() {
	c.client.Close()
}

// GetChainID 获取客户端连接的区块链网络ID
// 
// 返回:
//   - *big.Int: 区块链网络ID
func (c *Client) GetChainID() *big.Int {
	return c.chainID
}

// GetTransactionByHash 通过交易哈希获取交易详情
// 
// 参数:
//   - txHash: 交易哈希
//
// 返回:
//   - *types.Transaction: 交易对象
//   - bool: 交易是否处于pending状态
//   - error: 查询过程中的错误
func (c *Client) GetTransactionByHash(txHash common.Hash) (*types.Transaction, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return c.client.TransactionByHash(ctx, txHash)
}

// GetTransactionReceipt 获取已确认交易的收据信息
// 
// 参数:
//   - txHash: 交易哈希
//
// 返回:
//   - *types.Receipt: 交易收据，包含状态、事件日志等信息
//   - error: 查询过程中的错误，如果交易未确认会返回"not found"错误
func (c *Client) GetTransactionReceipt(txHash common.Hash) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return c.client.TransactionReceipt(ctx, txHash)
}
