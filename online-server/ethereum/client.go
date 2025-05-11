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

// ClientConfig 以太坊客户端配置
type ClientConfig struct {
	RPC         string
	ChainID     *big.Int
	ConfirmTime time.Duration // 等待交易确认的时间
	RetryTime   time.Duration // 重试间隔
	MaxRetries  int           // 最大重试次数
}

// DefaultConfig 返回默认配置
func DefaultConfig() ClientConfig {
	return ClientConfig{
		RPC:         "https://sepolia.infura.io/v3/766c230ed91a48a097e2739b966bbbf7",
		ChainID:     big.NewInt(11155111), // Sepolia 测试网
		ConfirmTime: 60 * time.Second,     // 等待交易确认的默认时间
		RetryTime:   10 * time.Second,     // 默认重试间隔
		MaxRetries:  5,                    // 默认最大重试次数
	}
}

// Client 以太坊客户端
type Client struct {
	client  *ethclient.Client
	config  ClientConfig
	chainID *big.Int
	mu      sync.Mutex // 用于并发访问的互斥锁
}

// NewClient 创建一个新的以太坊客户端
func NewClient(config ClientConfig) (*Client, error) {
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

// GetBalance 获取地址余额
func (c *Client) GetBalance(address string) (*big.Float, error) {
	addr := common.HexToAddress(address)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	balance, err := c.client.BalanceAt(ctx, addr, nil)
	if err != nil {
		return nil, fmt.Errorf("获取余额失败: %w", err)
	}

	fbalance := new(big.Float)
	fbalance.SetString(balance.String())
	ethValue := new(big.Float).Quo(fbalance, big.NewFloat(1e18))
	return ethValue, nil
}

// GetNonce 获取账户的 nonce
func (c *Client) GetNonce(address string) (uint64, error) {
	addr := common.HexToAddress(address)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return c.client.PendingNonceAt(ctx, addr)
}

// SuggestGasPrice 获取推荐 gas 价格
func (c *Client) SuggestGasPrice() (*big.Int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return c.client.SuggestGasPrice(ctx)
}

// SendTransaction 发送交易
func (c *Client) SendTransaction(tx *types.Transaction) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return c.client.SendTransaction(ctx, tx)
}

// WaitForConfirmation 等待交易确认
func (c *Client) WaitForConfirmation(tx *types.Transaction) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.config.ConfirmTime)
	defer cancel()

	return bind.WaitMined(ctx, c.client, tx)
}

// Close 关闭客户端连接
func (c *Client) Close() {
	c.client.Close()
}

// GetChainID 获取当前链ID
func (c *Client) GetChainID() *big.Int {
	return c.chainID
}

// GetTransactionByHash 通过交易哈希获取交易详情
func (c *Client) GetTransactionByHash(txHash common.Hash) (*types.Transaction, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return c.client.TransactionByHash(ctx, txHash)
}

// GetTransactionReceipt 获取交易收据
func (c *Client) GetTransactionReceipt(txHash common.Hash) (*types.Receipt, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return c.client.TransactionReceipt(ctx, txHash)
}

// NewEthClient 创建一个默认配置的以太坊客户端
func NewEthClient() (*Client, error) {
	return NewClient(DefaultConfig())
}
