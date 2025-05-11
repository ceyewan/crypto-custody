package ethereum

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var (
	serviceInstance     *Service
	serviceInstanceOnce sync.Once
)

// Service 以太坊服务，提供交易相关功能
type Service struct {
	client *Client
	mu     sync.RWMutex
	txData map[string]txPackage // 存储消息哈希到交易信息的映射
}

// txPackage 存储交易相关数据
type txPackage struct {
	nonce    uint64
	to       common.Address
	value    *big.Int
	gasLimit uint64
	gasPrice *big.Int
	data     []byte
	from     common.Address
}

// GetInstance 获取以太坊服务实例
func GetInstance() (*Service, error) {
	var initErr error

	serviceInstanceOnce.Do(func() {
		client, err := GetClientInstance()
		if err != nil {
			initErr = fmt.Errorf("无法初始化以太坊客户端: %w", err)
			return
		}

		serviceInstance = &Service{
			client: client,
			txData: make(map[string]txPackage),
		}
	})

	if initErr != nil {
		return nil, initErr
	}

	return serviceInstance, nil
}

// GetBalance 获取指定地址的余额
func (s *Service) GetBalance(address string) (*big.Float, error) {
	return s.client.GetBalance(address)
}

// PrepareTransaction 准备交易数据
func (s *Service) PrepareTransaction(from, to string, amount *big.Float) (string, error) {
	// 验证地址格式
	if !common.IsHexAddress(from) || !common.IsHexAddress(to) {
		return "", errors.New("无效的以太坊地址格式")
	}

	fromAddr := common.HexToAddress(from)
	toAddr := common.HexToAddress(to)

	// 获取nonce
	nonce, err := s.client.GetNonce(from)
	if err != nil {
		return "", fmt.Errorf("获取nonce失败: %w", err)
	}

	// 转换ETH为Wei
	value := new(big.Int)
	weiAmount := new(big.Float).Mul(amount, big.NewFloat(1e18))
	weiAmount.Int(value)

	// 获取Gas价格
	gasPrice, err := s.client.SuggestGasPrice()
	if err != nil {
		return "", fmt.Errorf("获取gas价格失败: %w", err)
	}

	// 标准ETH转账的gasLimit
	gasLimit := uint64(21000)

	// 创建交易
	tx := types.NewTransaction(
		nonce,
		toAddr,
		value,
		gasLimit,
		gasPrice,
		nil,
	)

	// 获取签名哈希
	signer := types.NewEIP155Signer(s.client.GetChainID())
	hash := signer.Hash(tx)

	// 存储交易信息以供后续使用
	s.mu.Lock()
	s.txData[hex.EncodeToString(hash[:])] = txPackage{
		nonce:    nonce,
		to:       toAddr,
		value:    value,
		gasLimit: gasLimit,
		gasPrice: gasPrice,
		data:     nil,
		from:     fromAddr,
	}
	s.mu.Unlock()

	return hex.EncodeToString(hash[:]), nil
}

// SignAndSendTransaction 使用签名发送交易
func (s *Service) SignAndSendTransaction(messageHash string, signature string) (string, error) {
	// 检查交易数据是否存在
	s.mu.RLock()
	txPkg, exists := s.txData[messageHash]
	s.mu.RUnlock()

	if !exists {
		return "", errors.New("未找到对应的交易数据，请先准备交易")
	}

	// 解码消息哈希
	hashBytes, err := hex.DecodeString(messageHash)
	if err != nil {
		return "", fmt.Errorf("解码消息哈希失败: %w", err)
	}

	// 解码签名
	sig, err := hexutil.Decode("0x" + signature)
	if err != nil {
		return "", fmt.Errorf("解码签名失败: %w", err)
	}

	// 恢复公钥
	pubKeyBytes, err := crypto.Ecrecover(hashBytes, sig)
	if err != nil {
		return "", fmt.Errorf("恢复公钥失败: %w", err)
	}

	pubKey, err := crypto.UnmarshalPubkey(pubKeyBytes)
	if err != nil {
		return "", fmt.Errorf("解析公钥失败: %w", err)
	}

	// 从公钥获取地址
	recoveredAddr := crypto.PubkeyToAddress(*pubKey)

	// 验证签名者地址与交易发送者地址是否匹配
	if recoveredAddr != txPkg.from {
		return "", errors.New("签名者地址与交易发送者地址不匹配")
	}

	// 重建交易
	tx := types.NewTransaction(
		txPkg.nonce,
		txPkg.to,
		txPkg.value,
		txPkg.gasLimit,
		txPkg.gasPrice,
		txPkg.data,
	)

	// 应用签名
	signer := types.NewEIP155Signer(s.client.GetChainID())
	signedTx, err := tx.WithSignature(signer, sig)
	if err != nil {
		return "", fmt.Errorf("应用签名失败: %w", err)
	}

	// 发送交易
	err = s.client.SendTransaction(signedTx)
	if err != nil {
		return "", fmt.Errorf("发送交易失败: %w", err)
	}

	// 返回交易哈希
	return signedTx.Hash().Hex(), nil
}

// Close 关闭服务
func (s *Service) Close() {
	if s.client != nil {
		s.client.Close()
	}
}
