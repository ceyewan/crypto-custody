package transaction_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const (
	// 基础URL
	BaseURL = "http://localhost:8080/api"

	// 测试用户数据
	TestUsername = "testuser"
	TestEmail    = "testuser@example.com"
	TestPassword = "testpassword123"

	// 管理员数据
	AdminUsername = "admin"
	AdminEmail    = "admin@example.com"

	// 测试账户数据
	TestEthAddress  = "0x742d35Cc6634C0532925a3b844Bc454e4438f44e"
	TestEthAddress2 = "0x8B3392483BA26D65E331dB86D4F430aE37546f4e"

	// 测试交易数据
	TestGasPrice = "20000000000" // 20 Gwei
	TestGasLimit = "21000"       // 标准ETH转账
	TestValue    = "0.001"       // 转账金额
	TestDataHex  = "0x"          // 空数据
)

// 应用返回的通用响应结构
type CommonResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// 用户信息结构
type UserInfo struct {
	ID        uint   `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	CreatedAt string `json:"createdAt,omitempty"`
}

// 登录响应数据
type LoginResponseData struct {
	Token string   `json:"token"`
	User  UserInfo `json:"user"`
}

// 账户信息
type AccountInfo struct {
	Address     string `json:"address"`
	CoinType    string `json:"coinType"`
	Balance     string `json:"balance"`
	ImportedBy  string `json:"importedBy"`
	Description string `json:"description"`
}

// 交易准备数据
type PrepareTransactionRequest struct {
	FromAddress string `json:"fromAddress"`
	ToAddress   string `json:"toAddress"`
	Value       string `json:"value"`
	GasPrice    string `json:"gasPrice"`
	GasLimit    string `json:"gasLimit"`
	Data        string `json:"data"`
	CoinType    string `json:"coinType"`
}

// 交易信息
type TransactionInfo struct {
	ID             string `json:"id"`
	Hash           string `json:"hash"`
	FromAddress    string `json:"fromAddress"`
	ToAddress      string `json:"toAddress"`
	Value          string `json:"value"`
	GasPrice       string `json:"gasPrice"`
	GasLimit       string `json:"gasLimit"`
	Nonce          uint64 `json:"nonce"`
	Data           string `json:"data"`
	Status         string `json:"status"`
	CoinType       string `json:"coinType"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
	RawTransaction string `json:"rawTransaction"`
	SignedData     string `json:"signedData"`
}

// 生成随机用户名, 避免测试冲突
func GenerateRandomUsername() string {
	return fmt.Sprintf("testuser_%d", time.Now().UnixNano())
}

// 生成随机邮箱
func GenerateRandomEmail() string {
	return fmt.Sprintf("testuser_%d@example.com", time.Now().UnixNano())
}

// 生成随机地址描述
func GenerateRandomDescription() string {
	return fmt.Sprintf("Test Account %d", time.Now().UnixNano())
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

	var loginData LoginResponseData
	if err := json.Unmarshal(response.Data, &loginData); err != nil {
		return "", err
	}

	return loginData.Token, nil
}

// 登录管理员并返回令牌
func LoginAdmin() (string, error) {
	// 从环境变量获取管理员密码
	adminPassword := os.Getenv("DEFAULT_ADMIN_PASSWORD")
	if adminPassword == "" {
		return "", fmt.Errorf("DEFAULT_ADMIN_PASSWORD environment variable not set")
	}

	return LoginUser(AdminUsername, adminPassword)
}

// 发送带认证的请求
func SendAuthenticatedRequest(method, url, token string, body []byte) (*http.Response, error) {
	client := &http.Client{}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", token)
	}

	return client.Do(req)
}

// 注册新用户
func RegisterNewUser(username, email, password string) (*UserInfo, error) {
	reqBody, err := json.Marshal(map[string]string{
		"username": username,
		"email":    email,
		"password": password,
	})
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(BaseURL+"/register", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if response.Code != 200 {
		return nil, fmt.Errorf("registration failed: %s", response.Message)
	}

	var user UserInfo
	if err := json.Unmarshal(response.Data, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

// 导入账户
func ImportAccount(token, address, coinType, description string) (*AccountInfo, error) {
	reqBody, err := json.Marshal(map[string]string{
		"address":     address,
		"coinType":    coinType,
		"description": description,
	})
	if err != nil {
		return nil, err
	}

	resp, err := SendAuthenticatedRequest("POST", BaseURL+"/accounts/import", token, reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if response.Code != 200 {
		return nil, fmt.Errorf("account import failed: %s", response.Message)
	}

	var account AccountInfo
	if err := json.Unmarshal(response.Data, &account); err != nil {
		return nil, err
	}

	return &account, nil
}

// 准备交易
func PrepareTransaction(token string, prepareReq PrepareTransactionRequest) (*TransactionInfo, error) {
	reqBody, err := json.Marshal(prepareReq)
	if err != nil {
		return nil, err
	}

	resp, err := SendAuthenticatedRequest("POST", BaseURL+"/transaction/prepare", token, reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if response.Code != 200 {
		return nil, fmt.Errorf("prepare transaction failed: %s", response.Message)
	}

	var transaction TransactionInfo
	if err := json.Unmarshal(response.Data, &transaction); err != nil {
		return nil, err
	}

	return &transaction, nil
}

// 提交签名交易
func SubmitSignedTransaction(token string, transactionID, signedData string) (*TransactionInfo, error) {
	reqBody, err := json.Marshal(map[string]string{
		"id":         transactionID,
		"signedData": signedData,
	})
	if err != nil {
		return nil, err
	}

	resp, err := SendAuthenticatedRequest("POST", BaseURL+"/transaction/submit", token, reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if response.Code != 200 {
		return nil, fmt.Errorf("submit transaction failed: %s", response.Message)
	}

	var transaction TransactionInfo
	if err := json.Unmarshal(response.Data, &transaction); err != nil {
		return nil, err
	}

	return &transaction, nil
}

// 获取交易详情
func GetTransactionByID(token, transactionID string) (*TransactionInfo, error) {
	resp, err := SendAuthenticatedRequest("GET", fmt.Sprintf("%s/transaction/%s", BaseURL, transactionID), token, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if response.Code != 200 {
		return nil, fmt.Errorf("get transaction failed: %s", response.Message)
	}

	var transaction TransactionInfo
	if err := json.Unmarshal(response.Data, &transaction); err != nil {
		return nil, err
	}

	return &transaction, nil
}

// 检查交易状态
func CheckTransactionStatus(token, transactionID string) (string, error) {
	transaction, err := GetTransactionByID(token, transactionID)
	if err != nil {
		return "", err
	}
	return transaction.Status, nil
}
