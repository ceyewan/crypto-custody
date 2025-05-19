package account_test

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

// 生成随机ETH地址
func GenerateRandomEthAddress() string {
	// ETH地址格式：0x + 40个十六进制字符
	const chars = "0123456789abcdef"
	addr := "0x"
	for i := 0; i < 40; i++ {
		addr += string(chars[time.Now().UnixNano()%int64(len(chars))])
		// 添加一点延迟确保每个字符都是不同的
		time.Sleep(time.Nanosecond)
	}
	return addr
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

// 创建账户
func CreateAccount(token, address, coinType, description string) (*AccountInfo, error) {
	reqBody, err := json.Marshal(map[string]string{
		"address":     address,
		"coinType":    coinType,
		"description": description,
	})
	if err != nil {
		return nil, err
	}

	resp, err := SendAuthenticatedRequest("POST", BaseURL+"/accounts/create", token, reqBody)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if response.Code != 200 {
		return nil, fmt.Errorf("account creation failed: %s", response.Message)
	}

	// 创建成功后通过地址查询账户信息
	return GetAccountByAddress(address)
}

// 导入账户
func ImportAccount(token, address, coinType, description string) (*AccountInfo, error) {
	// 构建批量导入请求的数据结构
	accounts := []map[string]string{
		{
			"address":     address,
			"coinType":    coinType,
			"description": description,
		},
	}

	importedAccounts, err := BatchImportAccounts(token, accounts)
	if err != nil {
		return nil, err
	}

	if len(importedAccounts) == 0 {
		return nil, fmt.Errorf("no account imported")
	}

	return &importedAccounts[0], nil
}

// 批量导入账户
func BatchImportAccounts(token string, accounts []map[string]string) ([]AccountInfo, error) {
	reqBody, err := json.Marshal(map[string]interface{}{
		"accounts": accounts,
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
		return nil, fmt.Errorf("batch import failed: %s", response.Message)
	}

	// 导入成功后获取所有账户并返回
	return GetAccounts(token)
}

// 获取账户通过地址
func GetAccountByAddress(address string) (*AccountInfo, error) {
	resp, err := http.Get(fmt.Sprintf("%s/accounts/address/%s", BaseURL, address))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if response.Code != 200 {
		return nil, fmt.Errorf("get account failed: %s", response.Message)
	}

	var account AccountInfo
	if err := json.Unmarshal(response.Data, &account); err != nil {
		return nil, err
	}

	return &account, nil
}

// 获取所有账户
func GetAccounts(token string) ([]AccountInfo, error) {
	resp, err := SendAuthenticatedRequest("GET", BaseURL+"/accounts", token, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if response.Code != 200 {
		return nil, fmt.Errorf("get accounts failed: %s", response.Message)
	}

	var accounts []AccountInfo
	if err := json.Unmarshal(response.Data, &accounts); err != nil {
		return nil, err
	}

	return accounts, nil
}

// 获取所有账户(管理员)
func GetAllAccounts(token string) ([]AccountInfo, error) {
	resp, err := SendAuthenticatedRequest("GET", BaseURL+"/accounts/admin/all", token, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	if response.Code != 200 {
		return nil, fmt.Errorf("get all accounts failed: %s", response.Message)
	}

	var result struct {
		Accounts []AccountInfo `json:"accounts"`
		Total    int           `json:"total"`
	}
	if err := json.Unmarshal(response.Data, &result); err != nil {
		return nil, err
	}

	return result.Accounts, nil
}
