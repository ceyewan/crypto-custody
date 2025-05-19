package transaction_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

const (
	// 基础URL
	BaseURL = "http://localhost:8080/api"

	// 测试账户地址 - 需要用户手动填写
	TestAccountA = "0x44C7d2CEC2ca6C8076188757529D73684088796d" // TODO: 请填写测试账户A的以太坊地址
	TestAccountB = "0xefd30a7F0A57edEF871872b53081d4057264fD72" // TODO: 请填写测试账户B的以太坊地址

	// 测试交易签名 - 需要用户手动填写
	TestSignature = "" // TODO: 请填写交易签名

	// 测试交易金额(ETH)
	TestAmount = 0.001
)

// 应用返回的通用响应结构
type CommonResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// 余额响应
type BalanceResponse struct {
	Address string `json:"address"`
	Balance string `json:"balance"`
}

// 准备交易响应
type PrepareTransactionResponse struct {
	TransactionID uint   `json:"transactionId"`
	MessageHash   string `json:"messageHash"`
}

// 交易发送响应
type SendTransactionResponse struct {
	TransactionID uint   `json:"transactionId"`
	TxHash        string `json:"txHash"`
}

// 执行登录并返回令牌
func LoginUser(username, password string) (string, error) {
	reqBody, err := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	if err != nil {
		return "", err
	}

	resp, err := http.Post(BaseURL+"/login", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	if response.Code != 200 {
		return "", fmt.Errorf("login failed: %s", response.Message)
	}

	var loginData struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(response.Data, &loginData); err != nil {
		return "", err
	}

	return loginData.Token, nil
}

// 登录警员并返回令牌
func LoginOfficer() (string, error) {
	// 从环境变量获取警员账户密码
	officerPassword := os.Getenv("OFFICER_PASSWORD")
	if officerPassword == "" {
		officerPassword = "officer123" // 默认密码，建议通过环境变量设置
	}

	return LoginUser("officer", officerPassword)
}
