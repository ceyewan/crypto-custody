package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"online-server/model"
	"online-server/utils"
	"sync"

	"github.com/ethereum/go-ethereum/core/types"
	"gorm.io/gorm"
)

var (
	transactionServiceInstance     *TransactionService
	transactionServiceInstanceOnce sync.Once
)

// TransactionService 提供交易数据的持久化存储和检索方法
type TransactionService struct {
	mu sync.RWMutex
}

// GetTransactionInstance 获取交易服务实例
func GetTransactionInstance() *TransactionService {
	transactionServiceInstanceOnce.Do(func() {
		transactionServiceInstance = &TransactionService{}
	})
	return transactionServiceInstance
}

// CreateTransaction 创建新的交易记录
//
// 参数:
//   - fromAddress: 发送方地址
//   - toAddress: 接收方地址
//   - value: 交易金额 (例如 "1.5 ETH")
//
// 返回:
//   - *model.Transaction: 成功创建的交易记录
//   - error: 创建过程中的错误
func (s *TransactionService) CreateTransaction(fromAddress, toAddress, value, messageHash string) (*model.Transaction, error) {
	// 验证必要字段
	if fromAddress == "" || toAddress == "" {
		return nil, errors.New("发送方和接收方地址不能为空")
	}

	if value == "" {
		return nil, errors.New("交易金额不能为空")
	}

	// 创建交易对象
	tx := &model.Transaction{
		FromAddress: fromAddress,
		ToAddress:   toAddress,
		Value:       value,
		MessageHash: messageHash,
		Status:      model.StatusPending, // 初始状态为Pending
	}

	result := utils.GetDB().Create(tx)
	if result.Error != nil {
		return nil, fmt.Errorf("创建交易记录失败: %w", result.Error)
	}

	return tx, nil
}

// GetTransactionByID 通过ID获取交易记录
//
// 参数:
//   - id: 交易记录ID
//
// 返回:
//   - *model.Transaction: 获取到的交易记录
//   - error: 查询过程中的错误
func (s *TransactionService) GetTransactionByID(id uint) (*model.Transaction, error) {
	var tx model.Transaction
	result := utils.GetDB().First(&tx, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("交易记录不存在")
		}
		return nil, fmt.Errorf("查询交易记录失败: %w", result.Error)
	}

	return &tx, nil
}

// GetTransactionByTxHash 通过交易哈希查询交易记录
//
// 参数:
//   - txHash: 交易哈希字符串(包含0x前缀)
//
// 返回:
//   - *model.Transaction: 获取到的交易记录
//   - error: 查询过程中的错误
func (s *TransactionService) GetTransactionByTxHash(txHash string) (*model.Transaction, error) {
	var tx model.Transaction
	result := utils.GetDB().Where("tx_hash = ?", txHash).First(&tx)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("未找到对应交易哈希的交易")
		}
		return nil, fmt.Errorf("查询交易记录失败: %w", result.Error)
	}

	return &tx, nil
}

// SignTransaction 为交易添加签名，并自动更新状态为Signed
//
// 参数:
//   - id: 要签名的交易记录ID
//   - signature: 签名数据
//
// 返回:
//   - *model.Transaction: 更新后的交易记录
//   - error: 更新过程中的错误
func (s *TransactionService) SignTransaction(id uint, signature []byte) (*model.Transaction, error) {
	tx, err := s.GetTransactionByID(id)
	if err != nil {
		return nil, err
	}

	// 检查交易状态
	if tx.Status != model.StatusPending {
		return nil, errors.New("只能为处于 Pending 状态的交易添加签名")
	}

	tx.Signature = signature
	tx.Status = model.StatusSigned // 自动更新状态为已签名

	result := utils.GetDB().Save(tx)
	if result.Error != nil {
		return nil, fmt.Errorf("更新交易签名失败: %w", result.Error)
	}

	return tx, nil
}

// SubmitTransaction 提交交易到网络，并自动更新状态为Submitted
//
// 参数:
//   - id: 要提交的交易记录ID
//   - txHash: 交易哈希(从区块链网络获取)
//
// 返回:
//   - *model.Transaction: 更新后的交易记录
//   - error: 更新过程中的错误
func (s *TransactionService) SubmitTransaction(id uint, txHash string) (*model.Transaction, error) {
	tx, err := s.GetTransactionByID(id)
	if err != nil {
		return nil, err
	}

	// 检查交易状态
	if tx.Status != model.StatusSigned {
		return nil, errors.New("只能提交处于 Signed 状态的交易")
	}

	// 更新交易哈希和状态
	tx.TxHash = txHash
	tx.Status = model.StatusSubmitted

	result := utils.GetDB().Save(tx)
	if result.Error != nil {
		return nil, fmt.Errorf("更新交易为提交状态失败: %w", result.Error)
	}

	return tx, nil
}

// UpdateTransactionReceipt 更新交易回执并根据回执状态自动更新为Confirmed或Failed状态
//
// 参数:
//   - id: 要更新的交易记录ID
//   - receipt: 交易回执数据
//
// 返回:
//   - *model.Transaction: 更新后的交易记录
//   - error: 更新过程中的错误
func (s *TransactionService) UpdateTransactionReceipt(id uint, receipt *types.Receipt) (*model.Transaction, error) {
	tx, err := s.GetTransactionByID(id)
	if err != nil {
		return nil, err
	}

	// 检查交易状态
	if tx.Status != model.StatusSubmitted {
		return nil, errors.New("只能为处于 Submitted 状态的交易更新回执")
	}

	// 序列化回执数据
	receiptJSON, err := receipt.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("序列化交易回执失败: %w", err)
	}

	tx.Receipt = receiptJSON

	// 根据回执状态自动更新交易状态
	if receipt.Status == types.ReceiptStatusSuccessful {
		tx.Status = model.StatusConfirmed
	} else {
		tx.Status = model.StatusFailed
	}

	result := utils.GetDB().Save(tx)
	if result.Error != nil {
		return nil, fmt.Errorf("更新交易回执失败: %w", result.Error)
	}

	return tx, nil
}

// GetPendingTransactions 获取所有未完成状态的交易
//
// 参数:
//   - limit: 查询的最大记录数
//   - offset: 分页偏移量
//
// 返回:
//   - []model.Transaction: 未完成的交易记录列表
//   - error: 查询过程中的错误
func (s *TransactionService) GetPendingTransactions(limit int, offset int) ([]model.Transaction, error) {
	var txs []model.Transaction

	if limit <= 0 {
		limit = 50 // 默认查询50条记录
	}

	result := utils.GetDB().Where("status IN ?", []model.TransactionStatus{
		model.StatusPending, model.StatusSigned, model.StatusSubmitted,
	}).
		Order("created_at ASC").
		Limit(limit).
		Offset(offset).
		Find(&txs)

	if result.Error != nil {
		return nil, fmt.Errorf("查询未完成交易记录失败: %w", result.Error)
	}

	return txs, nil
}

// GetTransactionsByStatus 根据状态查询交易
//
// 参数:
//   - status: 要查询的交易状态
//   - limit: 查询的最大记录数
//   - offset: 分页偏移量
//
// 返回:
//   - []model.Transaction: 指定状态的交易记录列表
//   - error: 查询过程中的错误
func (s *TransactionService) GetTransactionsByStatus(status model.TransactionStatus, limit int, offset int) ([]model.Transaction, error) {
	var txs []model.Transaction

	if limit <= 0 {
		limit = 50 // 默认查询50条记录
	}

	result := utils.GetDB().Where("status = ?", status).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&txs)

	if result.Error != nil {
		return nil, fmt.Errorf("查询交易记录失败: %w", result.Error)
	}

	return txs, nil
}

// ParseTransactionReceipt 将存储的交易回执数据解析为结构体
//
// 参数:
//   - tx: 包含回执数据的交易记录
//
// 返回:
//   - *types.Receipt: 解析后的交易回执结构体
//   - error: 解析过程中的错误
func (s *TransactionService) ParseTransactionReceipt(tx *model.Transaction) (*types.Receipt, error) {
	if tx == nil || len(tx.Receipt) == 0 {
		return nil, errors.New("交易回执数据为空")
	}

	var receipt types.Receipt
	err := json.Unmarshal(tx.Receipt, &receipt)
	if err != nil {
		return nil, fmt.Errorf("解析交易回执JSON失败: %w", err)
	}

	return &receipt, nil
}
