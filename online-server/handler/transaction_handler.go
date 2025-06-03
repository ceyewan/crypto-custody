package handler

import (
	"math/big"
	"net/http"
	"online-server/dto"
	"online-server/ethereum"
	"online-server/model"
	"online-server/service"
	"online-server/utils"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// 获取ETH余额
func GetBalance(c *gin.Context) {
	address := c.Param("address")
	if address == "" {
		utils.ResponseWithError(c, http.StatusBadRequest, "地址参数不能为空")
		return
	}

	// 获取客户端实例
	client, err := ethereum.GetClientInstance()
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "获取以太坊客户端失败："+err.Error())
		return
	}

	// 调用客户端的GetBalance方法
	balance, err := client.GetBalance(address)
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "获取余额失败："+err.Error())
		return
	}

	// 格式化余额为字符串，保留18位小数
	balanceStr := balance.Text('f', 18)

	utils.ResponseWithData(c, "获取余额成功", dto.BalanceResponse{
		Address: address,
		Balance: balanceStr,
	})
}

// 准备交易
func PrepareTransaction(c *gin.Context) {
	var req dto.PrepareTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseWithError(c, http.StatusBadRequest, "请求参数错误："+err.Error())
		return
	}

	// 获取交易管理器
	tm, err := getTransactionManager()
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "获取交易管理器失败："+err.Error())
		return
	}

	// 将float64转换为big.Float
	amount := new(big.Float).SetFloat64(req.Amount)

	// 创建交易
	txID, messageHash, err := tm.CreateTransaction(req.FromAddress, req.ToAddress, amount)
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "创建交易失败："+err.Error())
		return
	}

	utils.ResponseWithData(c, "交易准备成功", dto.PrepareTransactionResponse{
		TransactionID: txID,
		MessageHash:   messageHash,
	})
}

// 签名并发送交易
func SignAndSendTransaction(c *gin.Context) {
	var req dto.SignSendTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ResponseWithError(c, http.StatusBadRequest, "请求参数错误："+err.Error())
		return
	}

	// 获取交易管理器
	tm, err := getTransactionManager()
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "获取交易管理器失败："+err.Error())
		return
	}

	// 步骤1: 使用签名处理交易
	txID, err := tm.SignTransaction(req.MessageHash, req.Signature)
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "签名交易失败："+err.Error())
		return
	}

	// 步骤2: 发送交易
	_, txHash, err := tm.SendTransaction(req.MessageHash)
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "发送交易失败："+err.Error())
		return
	}

	utils.ResponseWithData(c, "交易签名并发送成功", dto.SendTransactionResponse{
		TransactionID: txID,
		TxHash:        txHash,
	})
}

// 获取交易管理器的辅助函数
func getTransactionManager() (*ethereum.TransactionManager, error) {
	return ethereum.GetTransactionManagerInstance()
}

// GetTransactionList 获取交易列表 (警员+)
func GetTransactionList(c *gin.Context) {
	// 从中间件中获取用户名
	username, exists := c.Get("Username")
	if !exists {
		utils.ResponseWithError(c, http.StatusUnauthorized, utils.ErrorUnauthorized)
		return
	}

	usernameStr, ok := username.(string)
	if !ok {
		utils.ResponseWithError(c, http.StatusInternalServerError, "用户信息类型错误")
		return
	}

	// 获取查询参数
	var req dto.TransactionListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.ResponseWithError(c, http.StatusBadRequest, "请求参数错误："+err.Error())
		return
	}

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	// 获取交易服务实例
	txService := service.GetTransactionInstance()

	// 查询与该用户相关的交易（发送方或接收方地址与用户导入的账户匹配）
	// 首先获取用户导入的账户
	accountService, err := service.GetAccountServiceInstance()
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "获取账户服务失败："+err.Error())
		return
	}

	userAccounts, err := accountService.GetAccountsByImportedBy(usernameStr)
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "获取用户账户失败："+err.Error())
		return
	}

	// 如果用户没有导入任何账户，返回空列表
	if len(userAccounts) == 0 {
		utils.ResponseWithData(c, "获取交易列表成功", dto.TransactionListResponse{
			Transactions: []dto.TransactionDetailResponse{},
			Total:        0,
			Page:         req.Page,
			PageSize:     req.PageSize,
		})
		return
	}

	// 构建地址列表用于查询
	var addressFilter string
	if req.Address != "" {
		// 检查指定地址是否属于该用户
		userOwnsAddress := false
		for _, account := range userAccounts {
			if strings.EqualFold(account.Address, req.Address) {
				userOwnsAddress = true
				break
			}
		}
		if !userOwnsAddress {
			utils.ResponseWithError(c, http.StatusForbidden, "无权查看该地址的交易")
			return
		}
		addressFilter = req.Address
	} else {
		// 如果没有指定地址，使用用户的第一个账户地址作为筛选
		if len(userAccounts) > 0 {
			addressFilter = userAccounts[0].Address
		}
	}

	// 查询交易列表
	transactions, total, err := txService.GetTransactionsList(req.Page, req.PageSize, req.Status, addressFilter)
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "查询交易列表失败："+err.Error())
		return
	}

	// 转换为响应格式
	var responseTransactions []dto.TransactionDetailResponse
	for _, tx := range transactions {
		statusText := getStatusText(tx.Status)
		statusString := getStatusString(tx.Status)
		responseTransactions = append(responseTransactions, dto.TransactionDetailResponse{
			ID:          tx.ID,
			FromAddress: tx.FromAddress,
			ToAddress:   tx.ToAddress,
			Value:       tx.Value,
			Amount:      tx.Value, // 添加amount字段给前端使用
			MessageHash: tx.MessageHash,
			TxHash:      tx.TxHash,
			Status:      statusString, // 使用字符串状态
			StatusText:  statusText,
			CreatedAt:   tx.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:   tx.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	utils.ResponseWithData(c, "获取交易列表成功", dto.TransactionListResponse{
		Transactions: responseTransactions,
		Total:        total,
		Page:         req.Page,
		PageSize:     req.PageSize,
	})
}

// GetAllTransactions 获取所有交易 (管理员)
func GetAllTransactions(c *gin.Context) {
	// 检查管理员权限
	if !utils.CheckAdminRole(c) {
		return
	}

	// 获取查询参数
	var req dto.TransactionListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.ResponseWithError(c, http.StatusBadRequest, "请求参数错误："+err.Error())
		return
	}

	// 设置默认值
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}

	// 获取交易服务实例
	txService := service.GetTransactionInstance()

	// 查询所有交易
	transactions, total, err := txService.GetTransactionsList(req.Page, req.PageSize, req.Status, req.Address)
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "查询交易列表失败："+err.Error())
		return
	}

	// 转换为响应格式
	var responseTransactions []dto.TransactionDetailResponse
	for _, tx := range transactions {
		statusText := getStatusText(tx.Status)
		statusString := getStatusString(tx.Status)
		responseTransactions = append(responseTransactions, dto.TransactionDetailResponse{
			ID:          tx.ID,
			FromAddress: tx.FromAddress,
			ToAddress:   tx.ToAddress,
			Value:       tx.Value,
			Amount:      tx.Value, // 添加amount字段给前端使用
			MessageHash: tx.MessageHash,
			TxHash:      tx.TxHash,
			Status:      statusString, // 使用字符串状态
			StatusText:  statusText,
			CreatedAt:   tx.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:   tx.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	utils.ResponseWithData(c, "获取所有交易成功", dto.TransactionListResponse{
		Transactions: responseTransactions,
		Total:        total,
		Page:         req.Page,
		PageSize:     req.PageSize,
	})
}

// GetTransactionByID 获取交易详情
func GetTransactionByID(c *gin.Context) {
	// 获取交易ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.ResponseWithError(c, http.StatusBadRequest, "无效的交易ID")
		return
	}

	// 获取交易服务实例
	txService := service.GetTransactionInstance()

	// 查询交易详情
	transaction, err := txService.GetTransactionByID(uint(id))
	if err != nil {
		utils.ResponseWithError(c, http.StatusNotFound, "交易不存在："+err.Error())
		return
	}

	// 转换为响应格式
	statusText := getStatusText(transaction.Status)
	statusString := getStatusString(transaction.Status)
	responseTransaction := dto.TransactionDetailResponse{
		ID:          transaction.ID,
		FromAddress: transaction.FromAddress,
		ToAddress:   transaction.ToAddress,
		Value:       transaction.Value,
		Amount:      transaction.Value, // 添加amount字段给前端使用
		MessageHash: transaction.MessageHash,
		TxHash:      transaction.TxHash,
		Status:      statusString, // 使用字符串状态
		StatusText:  statusText,
		CreatedAt:   transaction.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   transaction.UpdatedAt.Format("2006-01-02 15:04:05"),
	}

	utils.ResponseWithData(c, "获取交易详情成功", responseTransaction)
}

// GetTransactionStats 获取交易统计 (警员+)
func GetTransactionStats(c *gin.Context) {
	// 从中间件中获取用户名
	username, exists := c.Get("Username")
	if !exists {
		utils.ResponseWithError(c, http.StatusUnauthorized, utils.ErrorUnauthorized)
		return
	}

	_, ok := username.(string)
	if !ok {
		utils.ResponseWithError(c, http.StatusInternalServerError, "用户信息类型错误")
		return
	}

	// 获取交易服务实例
	txService := service.GetTransactionInstance()

	// 获取用户相关的交易统计
	// 这里简化处理，返回全局统计
	// 实际应用中可以根据用户权限筛选统计数据
	stats, err := txService.GetTransactionStats()
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "获取交易统计失败："+err.Error())
		return
	}

	utils.ResponseWithData(c, "获取交易统计成功", stats)
}

// GetAllTransactionStats 获取所有交易统计 (管理员)
func GetAllTransactionStats(c *gin.Context) {
	// 检查管理员权限
	if !utils.CheckAdminRole(c) {
		return
	}

	// 获取交易服务实例
	txService := service.GetTransactionInstance()

	// 获取全局交易统计
	stats, err := txService.GetTransactionStats()
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "获取交易统计失败："+err.Error())
		return
	}

	utils.ResponseWithData(c, "获取所有交易统计成功", stats)
}

// getStatusText 获取状态的文本描述
func getStatusText(status model.TransactionStatus) string {
	switch status {
	case model.StatusPending:
		return "待处理"
	case model.StatusSigned:
		return "已签名"
	case model.StatusSubmitted:
		return "已提交"
	case model.StatusConfirmed:
		return "已确认"
	case model.StatusFailed:
		return "失败"
	default:
		return "未知状态"
	}
}

// getStatusString 获取状态的字符串表示（前端期望格式）
func getStatusString(status model.TransactionStatus) string {
	switch status {
	case model.StatusPending:
		return "prepared"
	case model.StatusSigned:
		return "signed"
	case model.StatusSubmitted:
		return "sent"
	case model.StatusConfirmed:
		return "confirmed"
	case model.StatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}
