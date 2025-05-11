package ethereum

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"online-server/model"
	"online-server/utils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	ErrTransactionNotFound    = errors.New("交易未找到")
	ErrTransactionAlreadySent = errors.New("交易已发送")
	ErrInvalidSignature       = errors.New("无效的签名")
	ErrTransactionInProgress  = errors.New("交易正在处理中")
)

// TransactionManager 处理以太坊交易的管理器
type TransactionManager struct {
	client  *Client
	txMutex sync.RWMutex                  // 用于保护交易映射
	txCache map[string]*types.Transaction // 临时缓存待处理的交易，key是消息哈希
	signers map[string]types.Signer       // 临时缓存每个交易对应的签名者，key是消息哈希
}

// NewTransactionManager 创建一个新的交易管理器
func NewTransactionManager(client *Client) *TransactionManager {
	return &TransactionManager{
		client:  client,
		txCache: make(map[string]*types.Transaction),
		signers: make(map[string]types.Signer),
	}
}

// CreateTransaction 创建一个新的转账交易并存储到数据库
func (tm *TransactionManager) CreateTransaction(fromAddress, toAddress string, amount *big.Float) (*model.Transaction, string, error) {
	// 1. 获取必要的交易参数
	nonce, err := tm.client.GetNonce(fromAddress)
	if err != nil {
		return nil, "", fmt.Errorf("获取 nonce 失败: %w", err)
	}

	gasPrice, err := tm.client.SuggestGasPrice()
	if err != nil {
		return nil, "", fmt.Errorf("获取 gas 价格失败: %w", err)
	}

	// 转换 ETH 金额为 wei (1 ETH = 10^18 wei)
	value := new(big.Int)
	amountInWei := new(big.Float).Mul(amount, big.NewFloat(1e18))
	amountInWei.Int(value)

	gasLimit := uint64(21000) // 标准 ETH 转账的 gas 限制

	// 2. 创建交易对象
	tx := types.NewTransaction(
		nonce,
		common.HexToAddress(toAddress),
		value,
		gasLimit,
		gasPrice,
		nil, // 无数据
	)

	// 3. 计算交易哈希（用于签名）
	signer := types.NewEIP155Signer(tm.client.GetChainID())
	hash := signer.Hash(tx)
	messageHash := hex.EncodeToString(hash[:])

	// 4. 序列化交易用于存储
	txJSON, err := json.Marshal(struct {
		Nonce    uint64
		To       string
		Value    string
		GasLimit uint64
		GasPrice string
		Data     []byte
		ChainID  string
	}{
		Nonce:    tx.Nonce(),
		To:       tx.To().Hex(),
		Value:    tx.Value().String(),
		GasLimit: tx.Gas(),
		GasPrice: tx.GasPrice().String(),
		Data:     tx.Data(),
		ChainID:  tm.client.GetChainID().String(),
	})
	if err != nil {
		return nil, "", fmt.Errorf("序列化交易失败: %w", err)
	}

	// 5. 创建交易记录
	txRecord := &model.Transaction{
		FromAddress:     fromAddress,
		ToAddress:       toAddress,
		Value:           amount.String() + " ETH",
		Nonce:           nonce,
		GasLimit:        gasLimit,
		GasPrice:        gasPrice.String(),
		Status:          model.StatusPending,
		MessageHash:     messageHash,
		TransactionJSON: txJSON,
	}

	// 6. 存储交易记录到数据库
	result := utils.GetDB().Create(txRecord)
	if result.Error != nil {
		return nil, "", fmt.Errorf("存储交易记录失败: %w", result.Error)
	}

	// 7. 缓存交易对象和签名者，以便后续使用
	tm.txMutex.Lock()
	tm.txCache[messageHash] = tx
	tm.signers[messageHash] = signer
	tm.txMutex.Unlock()

	return txRecord, messageHash, nil
}

// SignTransaction 使用提供的签名处理交易
func (tm *TransactionManager) SignTransaction(messageHash string, signature string) (*model.Transaction, error) {
	// 1. 查找交易记录
	var tx model.Transaction
	result := utils.GetDB().Where("message_hash = ?", messageHash).First(&tx)
	if result.Error != nil {
		return nil, ErrTransactionNotFound
	}

	// 2. 检查交易状态
	if tx.Status != model.StatusPending {
		return nil, ErrTransactionAlreadySent
	}

	// 3. 解码签名
	signatureBytes, err := hex.DecodeString(signature)
	if err != nil {
		return nil, fmt.Errorf("解码签名失败: %w", err)
	}

	// 4. 获取缓存的交易和签名者
	tm.txMutex.RLock()
	rawTx, ok := tm.txCache[messageHash]
	signer, signerOk := tm.signers[messageHash]
	tm.txMutex.RUnlock()

	// 如果缓存中没有找到交易，则尝试从数据库中重构
	if !ok || !signerOk {
		var err error
		rawTx, signer, err = tm.reconstructTransaction(&tx)
		if err != nil {
			return nil, fmt.Errorf("重构交易失败: %w", err)
		}
	}

	// 5. 为交易附加签名
	signedTx, err := rawTx.WithSignature(signer, signatureBytes)
	if err != nil {
		return nil, fmt.Errorf("附加签名失败: %w", err)
	}

	// 6. 验证签名有效性
	sender, err := types.Sender(signer, signedTx)
	if err != nil {
		return nil, fmt.Errorf("验证签名失败: %w", err)
	}

	if sender.Hex() != tx.FromAddress {
		return nil, ErrInvalidSignature
	}

	// 7. 更新交易记录
	tx.Status = model.StatusSigned
	tx.Signature = signatureBytes
	tx.TxHash = signedTx.Hash().Hex()
	result = utils.GetDB().Save(&tx)
	if result.Error != nil {
		return nil, fmt.Errorf("更新交易记录失败: %w", result.Error)
	}

	// 8. 更新缓存
	tm.txMutex.Lock()
	tm.txCache[messageHash] = signedTx
	tm.txMutex.Unlock()

	return &tx, nil
}

// SendTransaction 发送已签名的交易到网络
func (tm *TransactionManager) SendTransaction(txID uint) (*model.Transaction, error) {
	// 1. 查找交易记录
	var tx model.Transaction
	result := utils.GetDB().First(&tx, txID)
	if result.Error != nil {
		return nil, ErrTransactionNotFound
	}

	// 2. 检查交易状态
	if tx.Status != model.StatusSigned {
		return nil, fmt.Errorf("交易状态不正确: %d", tx.Status)
	}

	// 3. 获取缓存的交易
	tm.txMutex.RLock()
	signedTx, ok := tm.txCache[tx.MessageHash]
	tm.txMutex.RUnlock()

	// 如果缓存中没有找到交易，则尝试从数据库中重构
	if !ok {
		var err error
		var signer types.Signer
		signedTx, signer, err = tm.reconstructTransaction(&tx)
		if err != nil {
			return nil, fmt.Errorf("重构交易失败: %w", err)
		}

		// 附加签名
		signedTx, err = signedTx.WithSignature(signer, tx.Signature)
		if err != nil {
			return nil, fmt.Errorf("附加签名失败: %w", err)
		}
	}

	// 4. 发送交易到网络
	now := time.Now()
	err := tm.client.SendTransaction(signedTx)
	if err != nil {
		// 如果是因为 nonce 太低导致的错误，可以增加 nonce 并重建交易
		// 这里简化处理，仅记录错误
		tx.Status = model.StatusFailed
		tx.Error = err.Error()
		utils.GetDB().Save(&tx)
		return nil, fmt.Errorf("发送交易失败: %w", err)
	}

	// 5. 更新交易记录
	tx.Status = model.StatusSubmitted
	tx.SubmittedAt = &now
	result = utils.GetDB().Save(&tx)
	if result.Error != nil {
		return nil, fmt.Errorf("更新交易记录失败: %w", result.Error)
	}

	// 6. 启动一个 goroutine 来监控交易状态
	go tm.monitorTransaction(signedTx.Hash(), tx.ID)

	return &tx, nil
}

// GetTransactionStatus 获取交易状态
func (tm *TransactionManager) GetTransactionStatus(txID uint) (*model.Transaction, error) {
	var tx model.Transaction
	result := utils.GetDB().First(&tx, txID)
	if result.Error != nil {
		return nil, ErrTransactionNotFound
	}
	return &tx, nil
}

// GetTransactionByMessageHash 通过消息哈希获取交易
func (tm *TransactionManager) GetTransactionByMessageHash(messageHash string) (*model.Transaction, error) {
	var tx model.Transaction
	result := utils.GetDB().Where("message_hash = ?", messageHash).First(&tx)
	if result.Error != nil {
		return nil, ErrTransactionNotFound
	}
	return &tx, nil
}

// GetPendingTransactions 获取所有待处理的交易
func (tm *TransactionManager) GetPendingTransactions() ([]model.Transaction, error) {
	var txs []model.Transaction
	result := utils.GetDB().Where("status IN (?)", []model.TransactionStatus{
		model.StatusPending, model.StatusSigned, model.StatusSubmitted,
	}).Find(&txs)
	if result.Error != nil {
		return nil, result.Error
	}
	return txs, nil
}

// GetUserTransactions 获取用户的所有交易
func (tm *TransactionManager) GetUserTransactions(address string) ([]model.Transaction, error) {
	var txs []model.Transaction
	result := utils.GetDB().Where("from_address = ? OR to_address = ?", address, address).
		Order("created_at DESC").Find(&txs)
	if result.Error != nil {
		return nil, result.Error
	}
	return txs, nil
}

// monitorTransaction 监控交易确认状态
func (tm *TransactionManager) monitorTransaction(txHash common.Hash, txID uint) {
	// 重试逻辑
	for i := 0; i < 3; i++ {
		// 等待一段时间后检查
		time.Sleep(tm.client.config.ConfirmTime)

		// 获取交易收据
		receipt, err := tm.client.GetTransactionReceipt(txHash)
		if err != nil {
			// 如果错误不是"未找到"，记录错误但继续重试
			if err.Error() != "not found" {
				fmt.Printf("获取交易收据失败: %v\n", err)
			}
			continue
		}

		// 交易已确认
		if receipt != nil {
			var tx model.Transaction
			result := utils.GetDB().First(&tx, txID)
			if result.Error != nil {
				fmt.Printf("获取交易记录失败: %v\n", result.Error)
				return
			}

			now := time.Now()
			tx.ConfirmedAt = &now
			tx.BlockNumber = receipt.BlockNumber.Uint64()
			tx.BlockHash = receipt.BlockHash.Hex()

			if receipt.Status == types.ReceiptStatusSuccessful {
				tx.Status = model.StatusConfirmed
			} else {
				tx.Status = model.StatusFailed
				tx.Error = "交易执行失败"
			}

			utils.GetDB().Save(&tx)
			return
		}
	}

	// 达到最大重试次数但仍未确认，标记为未确认状态
	var tx model.Transaction
	result := utils.GetDB().First(&tx, txID)
	if result.Error != nil {
		fmt.Printf("获取交易记录失败: %v\n", result.Error)
		return
	}

	// 虽然网络未确认，但仍保持已提交状态，后续可以手动检查
	tx.Error = "交易未在指定时间内确认，但可能稍后会被确认"
	utils.GetDB().Save(&tx)
}

// reconstructTransaction 从数据库记录重构交易对象
func (tm *TransactionManager) reconstructTransaction(tx *model.Transaction) (*types.Transaction, types.Signer, error) {
	// 解析存储的交易数据
	var txData struct {
		Nonce    uint64
		To       string
		Value    string
		GasLimit uint64
		GasPrice string
		Data     []byte
		ChainID  string
	}

	if err := json.Unmarshal(tx.TransactionJSON, &txData); err != nil {
		return nil, nil, fmt.Errorf("解析交易JSON失败: %w", err)
	}

	// 恢复 big.Int 值
	value := new(big.Int)
	value.SetString(txData.Value, 10)

	gasPrice := new(big.Int)
	gasPrice.SetString(txData.GasPrice, 10)

	chainID := new(big.Int)
	chainID.SetString(txData.ChainID, 10)

	// 重建交易
	rawTx := types.NewTransaction(
		txData.Nonce,
		common.HexToAddress(txData.To),
		value,
		txData.GasLimit,
		gasPrice,
		txData.Data,
	)

	// 创建签名者
	signer := types.NewEIP155Signer(chainID)

	return rawTx, signer, nil
}

// CheckPendingTransactions 检查所有待处理交易的状态
func (tm *TransactionManager) CheckPendingTransactions() error {
	var txs []model.Transaction
	// 查找所有已提交但未确认的交易
	result := utils.GetDB().Where("status = ?", model.StatusSubmitted).Find(&txs)
	if result.Error != nil {
		return result.Error
	}

	for _, tx := range txs {
		// 跳过没有交易哈希的记录
		if tx.TxHash == "" {
			continue
		}

		// 检查交易状态
		txHash := common.HexToHash(tx.TxHash)
		receipt, err := tm.client.GetTransactionReceipt(txHash)
		if err != nil {
			// 如果错误不是"未找到"，记录错误但继续处理下一个交易
			if err.Error() != "not found" {
				fmt.Printf("获取交易收据失败: %v\n", err)
			}
			continue
		}

		// 交易已确认
		if receipt != nil {
			now := time.Now()
			tx.ConfirmedAt = &now
			tx.LastCheckedAt = &now
			tx.BlockNumber = receipt.BlockNumber.Uint64()
			tx.BlockHash = receipt.BlockHash.Hex()

			if receipt.Status == types.ReceiptStatusSuccessful {
				tx.Status = model.StatusConfirmed
			} else {
				tx.Status = model.StatusFailed
				tx.Error = "交易执行失败"
			}

			utils.GetDB().Save(&tx)
		} else {
			// 交易未确认，更新最后检查时间
			now := time.Now()
			tx.LastCheckedAt = &now
			tx.RetryCount++
			utils.GetDB().Save(&tx)
		}
	}

	return nil
}

// IsTransactionInProgress 检查用户是否有正在处理中的交易
func (tm *TransactionManager) IsTransactionInProgress(fromAddress string) (bool, error) {
	var count int64
	result := utils.GetDB().Model(&model.Transaction{}).
		Where("from_address = ? AND status IN (?)",
			fromAddress,
			[]model.TransactionStatus{model.StatusPending, model.StatusSigned, model.StatusSubmitted}).
		Count(&count)

	if result.Error != nil {
		return false, result.Error
	}

	return count > 0, nil
}
