package service

import (
	"fmt"
	"math/big"
	"online-server/dto"
	"online-server/ethereum"
	"sync"
)

var (
	ethServiceInstance     *EthService
	ethServiceInstanceOnce sync.Once
)

// EthService 提供以太坊相关的服务
type EthService struct {
	ethService *ethereum.Service
}

// GetEthServiceInstance 获取以太坊服务实例
func GetEthServiceInstance() (*EthService, error) {
	var initErr error

	ethServiceInstanceOnce.Do(func() {
		ethSvc, err := ethereum.GetInstance()
		if err != nil {
			initErr = fmt.Errorf("无法初始化以太坊服务: %w", err)
			return
		}

		ethServiceInstance = &EthService{
			ethService: ethSvc,
		}
	})

	if initErr != nil {
		return nil, initErr
	}

	return ethServiceInstance, nil
}

// GetBalance 获取指定地址的ETH余额
func (s *EthService) GetBalance(address string) (*dto.BalanceResponse, error) {
	balance, err := s.ethService.GetBalance(address)
	if err != nil {
		return nil, fmt.Errorf("获取余额失败: %w", err)
	}

	return &dto.BalanceResponse{
		Address: address,
		Balance: balance.Text('f', 18),
		Symbol:  "ETH",
	}, nil
}

// PrepareTransaction 准备交易数据
func (s *EthService) PrepareTransaction(req *dto.TransactionRequest) (string, error) {
	amount := new(big.Float)
	amount.SetFloat64(req.Amount)

	messageHash, err := s.ethService.PrepareTransaction(req.FromAddress, req.ToAddress, amount)
	if err != nil {
		return "", fmt.Errorf("准备交易失败: %w", err)
	}

	return messageHash, nil
}

// SignAndSendTransaction 签名并发送交易
func (s *EthService) SignAndSendTransaction(req *dto.SignatureRequest) (string, error) {
	txHash, err := s.ethService.SignAndSendTransaction(req.MessageHash, req.Signature)
	if err != nil {
		return "", fmt.Errorf("签名和发送交易失败: %w", err)
	}

	return txHash, nil
}

// Close 关闭服务
func (s *EthService) Close() {
	if s.ethService != nil {
		s.ethService.Close()
	}
}
