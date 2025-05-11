package handlers

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"net/http"

	"online-server/model"
	"online-server/servers"
	"online-server/utils"

	"github.com/ceyewan/clog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
)

func GetAccounts(c *gin.Context) {
	logger := clog.Module("account")
	logger.Info("获取账户列表", clog.String("requester", c.GetString("Username")))
	
	var accounts []model.Account
	result := utils.GetDB().Find(&accounts)
	if result.Error != nil {
		logger.Error("查询账户列表失败", clog.Err(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	
	logger.Info("获取账户列表成功", clog.Int("count", len(accounts)))
	c.JSON(http.StatusOK, accounts)
}

// GetAccountBalance 获取账户余额
func GetAccountBalance(c *gin.Context) {
	logger := clog.Module("account")
	address := c.Param("address")
	
	logger.Info("获取账户余额请求", clog.String("address", address))
	
	if address == "" {
		logger.Warn("获取账户余额失败：地址为空")
		c.JSON(http.StatusBadRequest, gin.H{"error": "地址不能为空"})
		return
	}

	// 检查地址格式
	if !common.IsHexAddress(address) {
		logger.Warn("获取账户余额失败：无效的地址格式", clog.String("address", address))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的以太坊地址格式"})
		return
	}

	balance := servers.GetBalance(address)
	logger.Info("获取账户余额成功", 
		clog.String("address", address), 
		clog.String("balance", balance.String()))
	
	c.JSON(http.StatusOK, gin.H{
		"address": address,
		"balance": balance.String() + " ETH",
	})
}

func CreateAccount(c *gin.Context) {
	logger := clog.Module("account")
	logger.Info("创建账户请求", clog.String("requester", c.GetString("Username")))
	
	var input struct {
		PublicKeyX string `json:"PublicKeyX" binding:"required"`
		PublicKeyY string `json:"PublickeyY" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Warn("创建账户失败：参数错误", clog.Err(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	xInt := new(big.Int)
	xInt.SetString(input.PublicKeyX, 16)
	yInt := new(big.Int)
	yInt.SetString(input.PublicKeyY, 16)
	pubKey := crypto.PubkeyToAddress(ecdsa.PublicKey{X: xInt, Y: yInt})

	logger.Info("从公钥创建地址", clog.String("address", pubKey.Hex()))
	
	account := model.Account{Address: pubKey.Hex(), Balance: "0.00 ETH"}
	result := utils.GetDB().Create(&account)
	if result.Error != nil {
		logger.Error("保存账户失败", clog.Err(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	
	logger.Info("创建账户成功", clog.String("address", account.Address))
	c.JSON(http.StatusOK, account)
}

func TransferAll(c *gin.Context) {
	logger := clog.Module("account")
	logger.Info("批量转账请求", clog.String("admin", c.GetString("Username")))
	
	var accounts []model.Account
	result := utils.GetDB().Find(&accounts)
	if result.Error != nil {
		logger.Error("获取账户列表失败", clog.Err(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	logger.Info("开始为所有账户执行转账", clog.Int("account_count", len(accounts)))
	
	for i, account := range accounts {
		logger.Info("处理账户转账", 
			clog.Int("index", i+1), 
			clog.Int("total", len(accounts)), 
			clog.String("address", account.Address))
			
		servers.Transfer(account.Address)
		utils.GetDB().Save(&account)
	}

	logger.Info("批量转账完成", clog.Int("processed", len(accounts)))
	c.JSON(http.StatusOK, gin.H{"message": "Transfer all accounts successfully"})
}

func UpdateBalance(c *gin.Context) {
	logger := clog.Module("account")
	logger.Info("更新所有账户余额请求", clog.String("admin", c.GetString("Username")))
	
	var accounts []model.Account
	result := utils.GetDB().Find(&accounts)
	if result.Error != nil {
		logger.Error("获取账户列表失败", clog.Err(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	
	logger.Info("开始更新账户余额", clog.Int("account_count", len(accounts)))
	
	for i, account := range accounts {
		balance := servers.GetBalance(account.Address)
		oldBalance := account.Balance
		account.Balance = balance.String() + " ETH"
		
		logger.Info("更新账户余额", 
			clog.Int("index", i+1), 
			clog.Int("total", len(accounts)),
			clog.String("address", account.Address), 
			clog.String("old_balance", oldBalance), 
			clog.String("new_balance", account.Balance))
			
		utils.GetDB().Save(&account)
	}
	
	logger.Info("所有账户余额更新完成", clog.Int("processed", len(accounts)))
	c.JSON(http.StatusOK, gin.H{"message": "Update all accounts successfully"})
}

func PackTransferData(c *gin.Context) {
	logger := clog.Module("account")
	logger.Info("打包交易数据请求", clog.String("requester", c.GetString("Username")))
	
	var input struct {
		From   string  `json:"from" binding:"required"`
		To     string  `json:"to" binding:"required"`
		Amount float64 `json:"amount" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Warn("打包交易数据失败：参数错误", clog.Err(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logger.Info("准备打包交易数据", 
		clog.String("from", input.From), 
		clog.String("to", input.To), 
		clog.Float64("amount", input.Amount))
		
	data := servers.PackTransferData(input.From, input.To, input.Amount)
	
	if data == "" {
		logger.Error("打包交易数据失败", 
			clog.String("from", input.From), 
			clog.String("to", input.To))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "打包交易数据失败"})
		return
	}
	
	logger.Info("交易数据打包成功", clog.String("hash", data))
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func SubmitTransaction(c *gin.Context) {
	logger := clog.Module("account")
	logger.Info("提交交易请求", clog.String("requester", c.GetString("Username")))
	
	var input struct {
		Signature string `json:"signature" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		logger.Warn("提交交易失败：参数错误", clog.Err(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	logger.Info("准备发送交易", clog.String("signature_length", fmt.Sprintf("%d", len(input.Signature))))
	
	err := servers.SendTransfer(input.Signature)
	if err != nil {
		logger.Error("发送交易失败", clog.Err(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	logger.Info("交易发送成功")
	c.JSON(http.StatusOK, gin.H{"message": "Transaction sent successfully"})
}