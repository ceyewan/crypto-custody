package account_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

// 自定义解码响应函数，处理服务端可能返回的数字类型数据
func decodeResponse(data []byte, target interface{}) error {
	// 首先检查数据是否为空
	if len(data) == 0 {
		return nil
	}

	// 尝试将JSON数据解码为目标结构
	err := json.Unmarshal(data, target)
	if err == nil {
		return nil
	}

	// 如果解码失败，检查是否因为服务端返回的是数字而不是对象或数组
	var number json.Number
	if err := json.Unmarshal(data, &number); err == nil {
		// 对于不需要实际处理数字值的情况，忽略错误，返回空
		return nil
	}

	// 其他类型的解码错误
	return err
}

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
)

// 应用返回的通用响应结构
type CommonResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
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
	if err := decodeResponse(response.Data, &loginData); err != nil {
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
	if err := decodeResponse(response.Data, &user); err != nil {
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
	if err := decodeResponse(response.Data, &account); err != nil {
		return nil, err
	}

	return &account, nil
}

// 批量导入账户
func BatchImportAccounts(token string, accounts []map[string]string) ([]AccountInfo, error) {
	reqBody, err := json.Marshal(map[string]interface{}{
		"accounts": accounts,
	})
	if err != nil {
		return nil, err
	}

	resp, err := SendAuthenticatedRequest("POST", BaseURL+"/accounts/batch-import", token, reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if response.Code != 200 {
		return nil, fmt.Errorf("batch import failed: %s", response.Message)
	}

	var imported []AccountInfo
	if err := decodeResponse(response.Data, &imported); err != nil {
		return nil, err
	}

	return imported, nil
}

// 获取账户余额
func GetAccountBalance(address string) (string, error) {
	resp, err := http.Get(fmt.Sprintf("%s/transaction/balance/%s", BaseURL, address))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	if response.Code != 200 {
		return "", fmt.Errorf("get balance failed: %s", response.Message)
	}

	var balanceResp struct {
		Address string `json:"address"`
		Balance string `json:"balance"`
	}
	if err := json.Unmarshal(response.Data, &balanceResp); err != nil {
		return "", err
	}

	return balanceResp.Balance, nil
}
