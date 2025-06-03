package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"online-server/model"
	"online-server/utils"
	"sync"
	"time"

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

// GetTransactionsByMessageHash 通过消息哈希查询交易记录
//
// 参数:
//   - messageHash: 消息哈希字符串
//
// 返回:
//   - *model.Transaction: 获取到的交易记录
//   - error: 查询过程中的错误
func (s *TransactionService) GetTransactionsByMessageHash(messageHash string) (*model.Transaction, error) {
	var tx model.Transaction
	result := utils.GetDB().Where("message_hash = ?", messageHash).First(&tx)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("未找到对应消息哈希的交易")
		}
		return nil, fmt.Errorf("查询交易记录失败: %w", result.Error)
	}

	return &tx, nil
}

// GetTransactionsList 获取交易列表，支持分页和筛选
//
// 参数:
//   - page: 页码（从1开始）
//   - pageSize: 每页大小
//   - status: 状态筛选（可选）
//   - address: 地址筛选（可选，匹配发送方或接收方）
//
// 返回:
//   - []model.Transaction: 交易记录列表
//   - int64: 总记录数
//   - error: 查询过程中的错误
func (s *TransactionService) GetTransactionsList(page, pageSize int, status string, address string) ([]model.Transaction, int64, error) {
	var txs []model.Transaction
	var total int64

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	// 构建查询条件
	query := utils.GetDB().Model(&model.Transaction{})

	// 状态筛选
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 地址筛选（匹配发送方或接收方）
	if address != "" {
		query = query.Where("from_address = ? OR to_address = ?", address, address)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询交易总数失败: %w", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&txs).Error; err != nil {
		return nil, 0, fmt.Errorf("查询交易列表失败: %w", err)
	}

	return txs, total, nil
}

// GetTransactionStats 获取交易统计信息
//
// 返回:
//   - map[string]interface{}: 包含各种统计数据的map
//   - error: 查询过程中的错误
func (s *TransactionService) GetTransactionStats() (map[string]interface{}, error) {
	var stats map[string]interface{} = make(map[string]interface{})

	// 总交易数
	var totalCount int64
	if err := utils.GetDB().Model(&model.Transaction{}).Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("查询总交易数失败: %w", err)
	}
	stats["totalCount"] = totalCount

	// 各状态交易数
	statusCounts := make(map[string]int64)
	var results []struct {
		Status int
		Count  int64
	}

	if err := utils.GetDB().Model(&model.Transaction{}).
		Select("status, count(*) as count").
		Group("status").
		Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("查询状态统计失败: %w", err)
	}

	// 初始化所有状态计数为0
	statusCounts["pendingCount"] = 0
	statusCounts["signedCount"] = 0
	statusCounts["submittedCount"] = 0
	statusCounts["confirmedCount"] = 0
	statusCounts["failedCount"] = 0

	// 填充实际统计数据
	for _, result := range results {
		switch model.TransactionStatus(result.Status) {
		case model.StatusPending:
			statusCounts["pendingCount"] = result.Count
		case model.StatusSigned:
			statusCounts["signedCount"] = result.Count
		case model.StatusSubmitted:
			statusCounts["submittedCount"] = result.Count
		case model.StatusConfirmed:
			statusCounts["confirmedCount"] = result.Count
		case model.StatusFailed:
			statusCounts["failedCount"] = result.Count
		}
	}

	// 将状态统计添加到总统计中
	for key, value := range statusCounts {
		stats[key] = value
	}

	// 今日交易数
	var todayCount int64
	today := fmt.Sprintf("%s%%", time.Now().Format("2006-01-02"))
	if err := utils.GetDB().Model(&model.Transaction{}).
		Where("created_at LIKE ?", today).
		Count(&todayCount).Error; err != nil {
		return nil, fmt.Errorf("查询今日交易数失败: %w", err)
	}
	stats["todayCount"] = todayCount

	// 本周交易数
	var weekCount int64
	weekStart := time.Now().AddDate(0, 0, -7)
	if err := utils.GetDB().Model(&model.Transaction{}).
		Where("created_at >= ?", weekStart).
		Count(&weekCount).Error; err != nil {
		return nil, fmt.Errorf("查询本周交易数失败: %w", err)
	}
	stats["weekCount"] = weekCount

	// 本月交易数
	var monthCount int64
	monthStart := time.Now().AddDate(0, -1, 0)
	if err := utils.GetDB().Model(&model.Transaction{}).
		Where("created_at >= ?", monthStart).
		Count(&monthCount).Error; err != nil {
		return nil, fmt.Errorf("查询本月交易数失败: %w", err)
	}
	stats["monthCount"] = monthCount

	// 总交易金额（仅计算已确认的交易）
	stats["totalValue"] = "0" // 默认值，实际计算需要解析value字段

	return stats, nil
}

// DeleteTransaction 删除交易记录
//
// 参数:
//   - id: 要删除的交易记录ID
//
// 返回:
//   - error: 删除过程中的错误
func (s *TransactionService) DeleteTransaction(id uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := utils.GetDB().Delete(&model.Transaction{}, id).Error; err != nil {
		return fmt.Errorf("删除交易记录失败: %w", err)
	}

	return nil
}
