// Package ethereum 提供与以太坊区块链交互的功能，实现了交易准备、签名和发送的完整流程。
// 该包支持在线-离线分离的交易模式，提高了私钥管理的安全性。
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
	// serviceInstance 存储Service的单例实例
	serviceInstance     *Service
	// serviceInstanceOnce 确保Service只被初始化一次
	serviceInstanceOnce sync.Once
)

// Service 实现以太坊服务层，提供交易相关功能
// 该服务负责管理交易准备、签名验证和交易发送的完整流程
type Service struct {
	client *Client         // 与以太坊网络交互的客户端
	mu     sync.RWMutex    // 用于保护并发访问的读写锁
	txData map[string]txPackage // 存储消息哈希到交易信息的映射
}

// txPackage 存储交易相关数据，用于临时保存待签名的交易信息
type txPackage struct {
	nonce    uint64          // 交易序号
	to       common.Address  // 接收方地址
	value    *big.Int        // 交易金额(Wei)
	gasLimit uint64          // 交易的gas上限
	gasPrice *big.Int        // gas价格(Wei)
	data     []byte          // 交易数据
	from     common.Address  // 发送方地址
}

// GetInstance 获取以太坊服务的单例实例，确保全局只有一个服务实例
//
// 返回:
//   - *Service: 服务实例
//   - error: 初始化过程中的错误
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

// GetBalance 获取指定以太坊地址的ETH余额
//
// 参数:
//   - address: 以太坊地址（十六进制字符串）
//
// 返回:
//   - *big.Float: 以ETH为单位的余额
//   - error: 查询过程中的错误
func (s *Service) GetBalance(address string) (*big.Float, error) {
	return s.client.GetBalance(address)
}

// PrepareTransaction 准备以太坊交易数据，生成待签名的消息哈希
//
// 参数:
//   - from: 发送方地址（十六进制字符串）
//   - to: 接收方地址（十六进制字符串）
//   - amount: 交易金额，以ETH为单位
//
// 返回:
//   - string: 消息哈希的十六进制字符串，用于签名
//   - error: 准备过程中的错误
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

// SignAndSendTransaction 使用签名发送之前准备的交易
//
// 参数:
//   - messageHash: 之前通过PrepareTransaction生成的消息哈希
//   - signature: 消息哈希对应的签名（十六进制字符串，不含0x前缀）
//
// 返回:
//   - string: 交易哈希（含0x前缀）
//   - error: 签名验证或交易发送过程中的错误
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

// Close 关闭服务并释放相关资源
// 应在应用程序退出前调用此方法以确保资源被正确释放
func (s *Service) Close() {
	if s.client != nil {
		s.client.Close()
	}
}
