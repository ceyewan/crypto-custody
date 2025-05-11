package service

import (
	"online-server/dto"
	"sync"
	"time"
)

var (
	accountServiceInstance     *AccountService
	accountServiceInstanceOnce sync.Once
)

// TransactionStatus 交易状态
type TransactionStatus string

const (
	// StatusPending 待处理状态
	StatusPending TransactionStatus = "pending"
	// StatusSigned 已签名状态
	StatusSigned TransactionStatus = "signed"
	// StatusSubmitted 已提交状态
	StatusSubmitted TransactionStatus = "submitted"
	// StatusConfirmed 已确认状态
	StatusConfirmed TransactionStatus = "confirmed"
	// StatusFailed 失败状态
	StatusFailed TransactionStatus = "failed"
)

// AccountService 账户相关服务
type AccountService struct {
	mu sync.RWMutex
}

// GetAccountServiceInstance 获取账户服务实例
func GetAccountServiceInstance() (*AccountService, error) {
	accountServiceInstanceOnce.Do(func() {
		accountServiceInstance = &AccountService{}
	})
	return accountServiceInstance, nil
}

// SaveTransaction 保存交易到数据库
func (s *AccountService) SaveTransaction(fromAddress, toAddress string, amount float64, messageHash string) (uint, error) {
	// 这里应该是实际的数据库操作，为了示例，我们只返回一个模拟的ID
	// 在实际实现中，您需要与数据库进行交互

	// 模拟的ID生成
	id := uint(time.Now().Unix())

	return id, nil
}

// UpdateTransactionStatus 更新交易状态
func (s *AccountService) UpdateTransactionStatus(txID uint, status TransactionStatus, txHash string) error {
	// 这里应该是实际的数据库操作，为了示例，我们只返回成功
	// 在实际实现中，您需要与数据库进行交互
	return nil
}

// GetTransaction 获取交易详情
func (s *AccountService) GetTransaction(txID uint) (*dto.TransactionResponse, error) {
	// 这里应该是实际的数据库操作，为了示例，我们返回一个模拟数据
	// 在实际实现中，您需要从数据库中查询

	return &dto.TransactionResponse{
		ID:          txID,
		FromAddress: "0xSenderAddress",
		ToAddress:   "0xReceiverAddress",
		Amount:      "1.0",
		Status:      string(StatusPending),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

// GetTransactionByMessageHash 通过消息哈希获取交易
func (s *AccountService) GetTransactionByMessageHash(messageHash string) (*dto.TransactionResponse, error) {
	// 这里应该是实际的数据库操作，为了示例，我们返回一个模拟数据
	// 在实际实现中，您需要从数据库中查询

	return &dto.TransactionResponse{
		ID:          123,
		MessageHash: messageHash,
		FromAddress: "0xSenderAddress",
		ToAddress:   "0xReceiverAddress",
		Amount:      "1.0",
		Status:      string(StatusPending),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

// GetUserTransactions 获取用户的交易历史
func (s *AccountService) GetUserTransactions(address string) ([]dto.TransactionResponse, error) {
	// 这里应该是实际的数据库操作，为了示例，我们返回一个模拟数据
	// 在实际实现中，您需要从数据库中查询

	return []dto.TransactionResponse{
		{
			ID:          123,
			FromAddress: address,
			ToAddress:   "0xReceiverAddress1",
			Amount:      "1.0",
			Status:      string(StatusConfirmed),
			TxHash:      "0xTransactionHash1",
			CreatedAt:   time.Now().Add(-24 * time.Hour),
			UpdatedAt:   time.Now().Add(-23 * time.Hour),
		},
		{
			ID:          124,
			FromAddress: address,
			ToAddress:   "0xReceiverAddress2",
			Amount:      "2.0",
			Status:      string(StatusPending),
			CreatedAt:   time.Now().Add(-1 * time.Hour),
			UpdatedAt:   time.Now().Add(-1 * time.Hour),
		},
	}, nil
}
