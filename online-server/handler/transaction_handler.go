package handler

import (
	"math/big"
	"net/http"
	"online-server/dto"
	"online-server/ethereum"
	"online-server/utils"

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
