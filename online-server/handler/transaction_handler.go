package handler

import (
	"math/big"
	"net/http"
	"online-server/ethereum"
	"online-server/response"

	"github.com/gin-gonic/gin"
)

// 请求和响应的数据传输对象
type (
	// 准备交易请求
	PrepareTransactionRequest struct {
		FromAddress string  `json:"fromAddress" binding:"required"`
		ToAddress   string  `json:"toAddress" binding:"required"`
		Amount      float64 `json:"amount" binding:"required,gt=0"`
	}

	// 签名并发送交易请求
	SignSendTransactionRequest struct {
		MessageHash string `json:"messageHash" binding:"required"`
		Signature   string `json:"signature" binding:"required"`
	}

	// 余额响应
	BalanceResponse struct {
		Address string `json:"address"`
		Balance string `json:"balance"`
	}

	// 预备交易响应
	PrepareTransactionResponse struct {
		TransactionID uint   `json:"transactionId"`
		MessageHash   string `json:"messageHash"`
	}

	// 交易发送响应
	SendTransactionResponse struct {
		TransactionID uint   `json:"transactionId"`
		TxHash        string `json:"txHash"`
	}
)

// 获取ETH余额
func GetBalance(c *gin.Context) {
	address := c.Param("address")
	if address == "" {
		response.Fail(c, http.StatusBadRequest, "地址参数不能为空")
		return
	}

	// 获取客户端实例
	client, err := ethereum.GetClientInstance()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "获取以太坊客户端失败："+err.Error())
		return
	}

	// 调用客户端的GetBalance方法
	balance, err := client.GetBalance(address)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "获取余额失败："+err.Error())
		return
	}

	// 格式化余额为字符串，保留18位小数
	balanceStr := balance.Text('f', 18)

	response.Success(c, BalanceResponse{
		Address: address,
		Balance: balanceStr,
	}, "获取余额成功")
}

// 准备交易
func PrepareTransaction(c *gin.Context) {
	var req PrepareTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "请求参数错误："+err.Error())
		return
	}

	// 获取交易管理器
	tm, err := getTransactionManager()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "获取交易管理器失败："+err.Error())
		return
	}

	// 将float64转换为big.Float
	amount := new(big.Float).SetFloat64(req.Amount)

	// 创建交易
	txID, messageHash, err := tm.CreateTransaction(req.FromAddress, req.ToAddress, amount)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "创建交易失败："+err.Error())
		return
	}

	response.Success(c, PrepareTransactionResponse{
		TransactionID: txID,
		MessageHash:   messageHash,
	}, "交易准备成功")
}

// 签名并发送交易
func SignAndSendTransaction(c *gin.Context) {
	var req SignSendTransactionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "请求参数错误："+err.Error())
		return
	}

	// 获取交易管理器
	tm, err := getTransactionManager()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "获取交易管理器失败："+err.Error())
		return
	}

	// 步骤1: 使用签名处理交易
	txID, err := tm.SignTransaction(req.MessageHash, req.Signature)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "签名交易失败："+err.Error())
		return
	}

	// 步骤2: 发送交易
	_, txHash, err := tm.SendTransaction(req.MessageHash)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "发送交易失败："+err.Error())
		return
	}

	response.Success(c, SendTransactionResponse{
		TransactionID: txID,
		TxHash:        txHash,
	}, "交易签名并发送成功")
}

// 获取交易管理器的辅助函数
func getTransactionManager() (*ethereum.TransactionManager, error) {
	return ethereum.GetTransactionManagerInstance()
}
