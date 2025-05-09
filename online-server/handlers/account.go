package handlers

import (
	"crypto/ecdsa"
	"log"
	"math/big"
	"net/http"

	"backend/models"
	"backend/servers"
	"backend/utils"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gin-gonic/gin"
)

func GetAccounts(c *gin.Context) {
	var accounts []models.Account
	result := utils.DB.Find(&accounts)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	// 输出 accounts
	log.Println(accounts)
	c.JSON(http.StatusOK, accounts)
}

func CreateAccount(c *gin.Context) {
	var input struct {
		PublicKeyX string `json:"PublicKeyX" binding:"required"`
		PublicKeyY string `json:"PublickeyY" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	xInt := new(big.Int)
	xInt.SetString(input.PublicKeyX, 16)
	yInt := new(big.Int)
	yInt.SetString(input.PublicKeyY, 16)
	pubKey := crypto.PubkeyToAddress(ecdsa.PublicKey{X: xInt, Y: yInt})

	account := models.Account{Address: pubKey.Hex(), Balance: "0.00 ETH"}
	result := utils.DB.Create(&account)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, account)
}

func TransferAll(c *gin.Context) {
	var accounts []models.Account
	result := utils.DB.Find(&accounts)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	for _, account := range accounts {
		servers.Transfer(account.Address)
		utils.DB.Save(&account)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Transfer all accounts successfully"})
}

func UpdateBalance(c *gin.Context) {
	var accounts []models.Account
	result := utils.DB.Find(&accounts)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	for _, account := range accounts {
		balance := servers.GetBalance(account.Address)
		account.Balance = balance.String() + "ETH"
		utils.DB.Save(&account)
	}
	c.JSON(http.StatusOK, gin.H{"message": "Update all accounts successfully"})
}

func PackTransferData(c *gin.Context) {
	var input struct {
		From   string  `json:"from" binding:"required"`
		To     string  `json:"to" binding:"required"`
		Amount float64 `json:"amount" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	data := servers.PackTransferData(input.From, input.To, input.Amount)
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func SendTransaction(c *gin.Context) {
	var input struct {
		Signature string `json:"signature" binding:"required"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := servers.SendTransfer(input.Signature)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Transaction sent successfully"})
}
