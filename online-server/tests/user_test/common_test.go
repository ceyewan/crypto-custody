package user_test

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
	Valid   *bool           `json:"valid,omitempty"` // 用于check-auth接口
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

// 生成随机用户名, 避免测试冲突
func GenerateRandomUsername() string {
	return fmt.Sprintf("testuser_%d", time.Now().UnixNano())
}

// 生成随机邮箱
func GenerateRandomEmail() string {
	return fmt.Sprintf("testuser_%d@example.com", time.Now().UnixNano())
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
