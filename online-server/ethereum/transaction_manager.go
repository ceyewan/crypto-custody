// Package ethereum 提供与以太坊区块链交互的功能，实现了交易准备、签名和发送的完整流程。
// 该包支持在线-离线分离的交易模式，提高了私钥管理的安全性。
package ethereum

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"online-server/model"
	"online-server/service"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	// ErrTransactionNotFound 交易记录在数据库中不存在
	ErrTransactionNotFound = errors.New("交易未找到")
	// ErrTransactionAlreadySent 交易已经发送过无法重新处理
	ErrTransactionAlreadySent = errors.New("交易已发送")
	// ErrInvalidSignature 提供的签名无效或与发送者不匹配
	ErrInvalidSignature = errors.New("无效的签名")
	// ErrTransactionInProgress 用户已有正在处理中的交易
	ErrTransactionInProgress = errors.New("交易正在处理中")
	// errReceiptNotAvailable 交易收据暂时不可用
	errReceiptNotAvailable = errors.New("交易收据暂不可用")
)

// TransactionManager 管理以太坊交易的全生命周期
// 负责交易的创建、签名验证、发送和状态监控，同时协调与TransactionService的交互
type TransactionManager struct {
	client      *Client                       // 以太坊客户端
	txService   *service.TransactionService   // 交易服务，用于持久化存储
	txMutex     sync.RWMutex                  // 用于保护交易映射的读写锁
	txCache     map[string]*types.Transaction // 临时缓存待处理的交易，key是消息哈希
	signers     map[string]types.Signer       // 临时缓存每个交易对应的签名者，key是消息哈希
	messageData map[string]MessageData        // 存储交易相关的额外数据，key是消息哈希
}

// MessageData 存储与交易消息相关的额外数据
type MessageData struct {
	FromAddress string   // 发送方地址
	ToAddress   string   // 接收方地址
	Value       *big.Int // 交易金额（以Wei为单位）
	Nonce       uint64   // 交易序号
	TransID     uint     // 数据库中的交易ID
}

// NewTransactionManager 创建一个新的交易管理器实例
//
// 参数:
//   - client: 已初始化的以太坊客户端
//
// 返回:
//   - *TransactionManager: 初始化的交易管理器
func NewTransactionManager(client *Client) *TransactionManager {
	return &TransactionManager{
		client:      client,
		txService:   service.GetTransactionInstance(),
		txCache:     make(map[string]*types.Transaction),
		signers:     make(map[string]types.Signer),
		messageData: make(map[string]MessageData),
	}
}

// GetTransactionManagerInstance 获取交易管理器的单例实例
//
// 返回:
//   - *TransactionManager: 交易管理器实例
//   - error: 初始化过程中的错误
func GetTransactionManagerInstance() (*TransactionManager, error) {
	client, err := GetClientInstance()
	if err != nil {
		return nil, fmt.Errorf("获取以太坊客户端失败: %w", err)
	}

	return NewTransactionManager(client), nil
}

// CreateTransaction 创建一个新的ETH转账交易并存储到数据库
//
// 参数:
//   - fromAddress: 发送方地址（十六进制字符串）
//   - toAddress: 接收方地址（十六进制字符串）
//   - amount: 转账金额，以ETH为单位
//
// 返回:
//   - uint: 创建的交易记录ID
//   - string: 消息哈希，用于签名
//   - error: 创建过程中的错误
func (tm *TransactionManager) CreateTransaction(fromAddress, toAddress string, amount *big.Float) (uint, string, error) {
	// 1. 检查用户是否有正在进行的交易
	inProgress, err := tm.IsTransactionInProgress(fromAddress)
	if err != nil {
		return 0, "", fmt.Errorf("检查交易状态失败: %w", err)
	}

	if inProgress {
		return 0, "", ErrTransactionInProgress
	}

	// 2. 获取必要的交易参数
	nonce, err := tm.client.GetNonce(fromAddress)
	if err != nil {
		return 0, "", fmt.Errorf("获取 nonce 失败: %w", err)
	}

	gasPrice, err := tm.client.SuggestGasPrice()
	if err != nil {
		return 0, "", fmt.Errorf("获取 gas 价格失败: %w", err)
	}

	// 3. 转换 ETH 金额为 wei (1 ETH = 10^18 wei)
	value := new(big.Int)
	amountInWei := new(big.Float).Mul(amount, big.NewFloat(1e18))
	amountInWei.Int(value)

	gasLimit := uint64(21000) // 标准 ETH 转账的 gas 限制

	// 4. 创建交易对象
	tx := types.NewTransaction(
		nonce,
		common.HexToAddress(toAddress),
		value,
		gasLimit,
		gasPrice,
		nil, // 无数据
	)

	// 5. 计算交易哈希（用于签名）
	signer := types.NewEIP155Signer(tm.client.GetChainID())
	hash := signer.Hash(tx)
	messageHash := hex.EncodeToString(hash[:])

	// 6. 创建交易记录
	amountEth := new(big.Float).SetInt(value)
	amountEth.Quo(amountEth, big.NewFloat(1e18))

	txRecord, err := tm.txService.CreateTransaction(
		fromAddress,
		toAddress,
		fmt.Sprintf("%s ETH", amountEth.Text('f', 18)),
		messageHash,
	)
	if err != nil {
		return 0, "", fmt.Errorf("存储交易记录失败: %w", err)
	}

	// 7. 缓存交易对象和签名者，以便后续使用
	tm.txMutex.Lock()
	tm.txCache[messageHash] = tx
	tm.signers[messageHash] = signer
	tm.messageData[messageHash] = MessageData{
		FromAddress: fromAddress,
		ToAddress:   toAddress,
		Value:       value,
		Nonce:       nonce,
		TransID:     txRecord.ID,
	}
	tm.txMutex.Unlock()

	return txRecord.ID, messageHash, nil
}

// SignTransaction 使用提供的签名处理交易
//
// 参数:
//   - messageHash: 交易的消息哈希（十六进制字符串，不含0x前缀）
//   - signature: 对消息哈希的签名（十六进制字符串，不含0x前缀）
//
// 返回:
//   - uint: 更新后的交易记录ID
//   - error: 签名处理过程中的错误
func (tm *TransactionManager) SignTransaction(messageHash string, signature string) (uint, error) {
	// 1. 获取消息数据
	tm.txMutex.RLock()
	messageData, ok := tm.messageData[messageHash]
	rawTx, txOk := tm.txCache[messageHash]
	signer, signerOk := tm.signers[messageHash]
	tm.txMutex.RUnlock()

	if !ok || !txOk || !signerOk {
		return 0, ErrTransactionNotFound
	}

	// 2. 解码签名
	signatureBytes, err := hex.DecodeString(signature)
	if err != nil {
		return 0, fmt.Errorf("解码签名失败: %w", err)
	}

	// 3. 为交易附加签名
	signedTx, err := rawTx.WithSignature(signer, signatureBytes)
	if err != nil {
		return 0, fmt.Errorf("附加签名失败: %w", err)
	}

	// 4. 验证签名有效性
	sender, err := types.Sender(signer, signedTx)
	if err != nil {
		return 0, fmt.Errorf("验证签名失败: %w", err)
	}

	if sender.Hex() != messageData.FromAddress {
		return 0, ErrInvalidSignature
	}

	// 5. 更新交易记录
	txID := messageData.TransID
	_, err = tm.txService.SignTransaction(txID, signatureBytes)
	if err != nil {
		return 0, fmt.Errorf("更新交易签名失败: %w", err)
	}

	// 6. 更新缓存
	tm.txMutex.Lock()
	tm.txCache[messageHash] = signedTx
	tm.txMutex.Unlock()

	return txID, nil
}

// SendTransaction 将已签名的交易发送到以太坊网络
//
// 参数:
//   - messageHash: 交易的消息哈希（十六进制字符串，不含0x前缀）
//
// 返回:
//   - uint: 已发送的交易记录ID
//   - string: 区块链网络返回的交易哈希
//   - error: 发送过程中的错误
func (tm *TransactionManager) SendTransaction(messageHash string) (uint, string, error) {
	// 1. 获取交易数据
	tm.txMutex.RLock()
	messageData, ok := tm.messageData[messageHash]
	signedTx, txOk := tm.txCache[messageHash]
	tm.txMutex.RUnlock()

	if !ok || !txOk {
		return 0, "", ErrTransactionNotFound
	}

	// 2. 发送交易到网络
	err := tm.client.SendTransaction(signedTx)
	if err != nil {
		return 0, "", fmt.Errorf("发送交易失败: %w", err)
	}

	txHash := signedTx.Hash().Hex()

	// 3. 更新交易记录
	txID := messageData.TransID
	_, err = tm.txService.SubmitTransaction(txID, txHash)
	if err != nil {
		return 0, "", fmt.Errorf("更新交易状态失败: %w", err)
	}

	// 4. 启动一个 goroutine 来监控交易状态
	go tm.monitorTransaction(signedTx.Hash(), txID)

	return txID, txHash, nil
}

// GetTransactionStatus 获取指定消息哈希交易的最新状态
//
// 参数:
//   - messageHash: 交易的消息哈希
//
// 返回:
//   - model.TransactionStatus: 交易当前状态
//   - error: 查询过程中的错误
func (tm *TransactionManager) GetTransactionStatus(messageHash string) (model.TransactionStatus, error) {
	tm.txMutex.RLock()
	messageData, ok := tm.messageData[messageHash]
	tm.txMutex.RUnlock()

	if !ok {
		return 0, ErrTransactionNotFound
	}

	txID := messageData.TransID
	tx, err := tm.txService.GetTransactionByID(txID)
	if err != nil {
		return 0, err
	}

	return tx.Status, nil
}

// GetTransactionByID 通过ID获取交易记录
//
// 参数:
//   - id: 交易记录ID
//
// 返回:
//   - *model.Transaction: 获取到的交易记录
//   - error: 查询过程中的错误
func (tm *TransactionManager) GetTransactionByID(id uint) (*model.Transaction, error) {
	return tm.txService.GetTransactionByID(id)
}

// GetMessageHashByID 通过交易ID获取消息哈希
//
// 参数:
//   - id: 交易记录ID
//
// 返回:
//   - string: 消息哈希
//   - error: 查询过程中的错误
func (tm *TransactionManager) GetMessageHashByID(id uint) (string, error) {
	tm.txMutex.RLock()
	defer tm.txMutex.RUnlock()

	for hash, data := range tm.messageData {
		if data.TransID == id {
			return hash, nil
		}
	}

	return "", ErrTransactionNotFound
}

// monitorTransaction 持续监控交易的确认状态并更新数据库
//
// 参数:
//   - txHash: 交易哈希
//   - txID: 交易记录的数据库ID
func (tm *TransactionManager) monitorTransaction(txHash common.Hash, txID uint) {
	// 重试逻辑
	for i := 0; i < 12; i++ {
		// 等待一段时间后检查
		time.Sleep(60 * time.Second)

		// 获取交易收据
		receipt, err := tm.client.GetTransactionReceipt(txHash)
		if err != nil {
			// 如果错误不是"未找到"，记录错误但继续重试
			if err.Error() != "not found" {
				fmt.Printf("Error checking transaction receipt: %v\n", err)
			}
			continue
		}

		// 交易已确认
		if receipt != nil {
			// 更新交易收据信息
			_, err := tm.txService.UpdateTransactionReceipt(txID, receipt)
			if err != nil {
				fmt.Printf("Error updating transaction receipt: %v\n", err)
			}
			return
		}
	}

	fmt.Printf("Transaction %s not confirmed after maximum retry attempts\n", txHash.Hex())
}

// IsTransactionInProgress 检查用户是否有正在处理中的交易
// 用于防止用户同时发起多笔待处理的交易
//
// 参数:
//   - fromAddress: 用户的以太坊地址
//
// 返回:
//   - bool: 是否存在处理中的交易
//   - error: 查询过程中的错误
func (tm *TransactionManager) IsTransactionInProgress(fromAddress string) (bool, error) {
	// 检查内存中是否有正在处理的交易
	tm.txMutex.RLock()
	for _, data := range tm.messageData {
		if data.FromAddress == fromAddress {
			// 查询交易状态
			tx, err := tm.txService.GetTransactionByID(data.TransID)
			if err != nil {
				tm.txMutex.RUnlock()
				return false, err
			}

			// 如果不是已确认或失败状态，则认为正在处理
			if tx.Status != model.StatusConfirmed && tx.Status != model.StatusFailed {
				tm.txMutex.RUnlock()
				return true, nil
			}
		}
	}
	tm.txMutex.RUnlock()

	// 查询数据库中的交易状态
	txs, err := tm.txService.GetPendingTransactions(1, 0)
	if err != nil {
		return false, err
	}

	for _, tx := range txs {
		if tx.FromAddress == fromAddress {
			return true, nil
		}
	}

	return false, nil
}

// CheckPendingTransactions 检查所有未确认交易的状态
// 该方法通常由定时任务调用，确保长时间未确认的交易得到处理
//
// 返回:
//   - error: 检查过程中的错误
func (tm *TransactionManager) CheckPendingTransactions() error {
	txs, err := tm.txService.GetPendingTransactions(100, 0)
	if err != nil {
		return fmt.Errorf("获取待处理交易失败: %w", err)
	}

	for _, tx := range txs {
		if tx.Status != model.StatusSubmitted || tx.TxHash == "" {
			continue
		}

		txHash := common.HexToHash(tx.TxHash)
		receipt, err := tm.client.GetTransactionReceipt(txHash)
		if err != nil {
			if err.Error() != "not found" {
				fmt.Printf("Error checking receipt for tx %s: %v\n", tx.TxHash, err)
			}
			continue
		}

		if receipt != nil {
			_, err := tm.txService.UpdateTransactionReceipt(tx.ID, receipt)
			if err != nil {
				fmt.Printf("Error updating receipt for tx %s: %v\n", tx.TxHash, err)
			}
		}
	}

	return nil
}

// ClearCompletedTransactions 清理已完成的交易缓存
// 该方法应定期调用以释放内存
func (tm *TransactionManager) ClearCompletedTransactions() {
	tm.txMutex.Lock()
	defer tm.txMutex.Unlock()

	for hash, data := range tm.messageData {
		tx, err := tm.txService.GetTransactionByID(data.TransID)
		if err != nil {
			continue
		}

		// 如果交易已确认或失败，从缓存中移除
		if tx.Status == model.StatusConfirmed || tx.Status == model.StatusFailed {
			delete(tm.txCache, hash)
			delete(tm.signers, hash)
			delete(tm.messageData, hash)
		}
	}
}

// GetTransactionCount 获取用户交易总数
//
// 参数:
//   - address: 用户地址
//
// 返回:
//   - int64: 交易总数
//   - error: 查询过程中的错误
func (tm *TransactionManager) GetTransactionCount(address string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 获取当前区块高度的nonce（表示已确认交易数）
	confirmedCount, err := tm.client.client.NonceAt(ctx, common.HexToAddress(address), nil)
	if err != nil {
		return 0, fmt.Errorf("获取已确认交易数失败: %w", err)
	}

	// 获取待处理交易数（包括内存池中的交易）
	pendingCount, err := tm.client.client.PendingNonceAt(ctx, common.HexToAddress(address))
	if err != nil {
		return 0, fmt.Errorf("获取待处理交易数失败: %w", err)
	}

	// 返回较大的值，通常是待处理交易数
	if pendingCount > confirmedCount {
		return int64(pendingCount), nil
	}
	return int64(confirmedCount), nil
}
