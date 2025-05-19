package service

import (
	"fmt"
	"online-server/model"
)

// GetTransaction 查询交易详情
func (s *AccountService) GetTransaction(id uint) (*model.Transaction, error) {
	var transaction model.Transaction
	result := s.db.First(&transaction, id)
	if result.Error != nil {
		return nil, fmt.Errorf("获取交易详情失败: %w", result.Error)
	}
	return &transaction, nil
}

// GetTransactionByMessageHash 通过消息哈希获取交易
func (s *AccountService) GetTransactionByMessageHash(messageHash string) (*model.Transaction, error) {
	var transaction model.Transaction
	result := s.db.Where("message_hash = ?", messageHash).First(&transaction)
	if result.Error != nil {
		return nil, fmt.Errorf("通过消息哈希获取交易失败: %w", result.Error)
	}
	return &transaction, nil
}

// GetUserTransactions 获取用户的交易历史
func (s *AccountService) GetUserTransactions(address string) ([]model.Transaction, error) {
	var transactions []model.Transaction
	result := s.db.Where("from_address = ? OR to_address = ?", address, address).Find(&transactions)
	if result.Error != nil {
		return nil, fmt.Errorf("获取用户交易历史失败: %w", result.Error)
	}
	return transactions, nil
}
