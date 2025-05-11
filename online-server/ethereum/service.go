package ethereum

import (
	"fmt"
	"math/big"
	"sync"

	"online-server/model"
)

var (
	ethService     *EthService
	ethServiceOnce sync.Once
)

// EthService 集成以太坊服务功能
type EthService struct {
	client            *Client
	transactionMgr    *TransactionManager
	isInitialized     bool
	initializationErr error
	mu                sync.RWMutex // 保护并发访问
}

// NewEthService 创建一个新的以太坊服务
func NewEthService() (*EthService, error) {
	client, err := NewEthClient()
	if err != nil {
		return nil, fmt.Errorf("创建以太坊客户端失败: %w", err)
	}

	txManager := NewTransactionManager(client)

	return &EthService{
		client:         client,
		transactionMgr: txManager,
		isInitialized:  true,
	}, nil
}

// GetInstance 获取EthService单例
func GetInstance() (*EthService, error) {
	ethServiceOnce.Do(func() {
		var err error
		ethService, err = NewEthService()
		if err != nil {
			ethService = &EthService{
				isInitialized:     false,
				initializationErr: err,
			}
		}
	})

	if !ethService.isInitialized {
		return nil, ethService.initializationErr
	}

	return ethService, nil
}

// GetBalance 获取账户余额
func (s *EthService) GetBalance(address string) (*big.Float, error) {
	return s.client.GetBalance(address)
}

// CreateTransaction 创建新交易
func (s *EthService) CreateTransaction(fromAddress, toAddress string, amount *big.Float) (*model.Transaction, string, error) {
	// 检查用户是否有未完成的交易
	inProgress, err := s.transactionMgr.IsTransactionInProgress(fromAddress)
	if err != nil {
		return nil, "", fmt.Errorf("检查用户交易状态失败: %w", err)
	}

	if inProgress {
		return nil, "", fmt.Errorf("用户有正在处理中的交易，请等待完成或检查交易状态")
	}

	return s.transactionMgr.CreateTransaction(fromAddress, toAddress, amount)
}

// SignTransaction 使用签名处理交易
func (s *EthService) SignTransaction(messageHash string, signature string) (*model.Transaction, error) {
	return s.transactionMgr.SignTransaction(messageHash, signature)
}

// SendTransaction 发送已签名的交易
func (s *EthService) SendTransaction(txID uint) (*model.Transaction, error) {
	return s.transactionMgr.SendTransaction(txID)
}

// GetTransactionStatus 获取交易状态
func (s *EthService) GetTransactionStatus(txID uint) (*model.Transaction, error) {
	return s.transactionMgr.GetTransactionStatus(txID)
}

// GetTransactionByMessageHash 通过消息哈希获取交易
func (s *EthService) GetTransactionByMessageHash(messageHash string) (*model.Transaction, error) {
	return s.transactionMgr.GetTransactionByMessageHash(messageHash)
}

// GetUserTransactions 获取用户的交易历史
func (s *EthService) GetUserTransactions(address string) ([]model.Transaction, error) {
	return s.transactionMgr.GetUserTransactions(address)
}

// CheckPendingTransactions 检查所有待处理的交易
func (s *EthService) CheckPendingTransactions() error {
	return s.transactionMgr.CheckPendingTransactions()
}

// Close 关闭以太坊服务
func (s *EthService) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		s.client.Close()
	}
}
