package user_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
)

func init() {
	// 加载 .env 文件
	rootDir := filepath.Join("..", "..") // 从 tests/user_test 目录回到项目根目录
	err := godotenv.Load(filepath.Join(rootDir, ".env"))
	if err != nil {
		// 仅打印错误，而不是退出，以便测试仍然可以运行
		println("警告: 无法加载 .env 文件:", err.Error())
	}
}

// 测试正确的登录凭据
func TestLoginWithValidCredentials(t *testing.T) {
	// 首先创建一个测试用户
	username := GenerateRandomUsername()
	email := GenerateRandomEmail()

	_, err := RegisterNewUser(username, email, TestPassword)
	if err != nil {
		t.Fatalf("注册测试用户失败: %v", err)
	}

	// 测试登录
	reqBody, _ := json.Marshal(map[string]string{
		"username": username,
		"password": TestPassword,
	})

	resp, err := http.Post(BaseURL+"/login", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("登录请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证响应状态码
	if resp.StatusCode != 200 {
		t.Errorf("预期状态码 200, 但收到 %d", resp.StatusCode)
	}

	// 解析响应内容
	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证响应内容
	if response.Code != 200 {
		t.Errorf("预期响应代码 200, 但收到 %d", response.Code)
	}

	if response.Message != "登录成功" {
		t.Errorf("预期消息 '登录成功', 但收到 '%s'", response.Message)
	}

	// 验证返回的用户数据和令牌
	var loginData LoginResponseData
	if err := json.Unmarshal(response.Data, &loginData); err != nil {
		t.Fatalf("解析用户数据失败: %v", err)
	}

	if loginData.Token == "" {
		t.Error("预期返回令牌, 但未收到")
	}

	if loginData.User.Username != username {
		t.Errorf("预期用户名 %s, 但收到 %s", username, loginData.User.Username)
	}

	if loginData.User.Email != email {
		t.Errorf("预期邮箱 %s, 但收到 %s", email, loginData.User.Email)
	}
}

// 测试错误的用户名
func TestLoginWithInvalidUsername(t *testing.T) {
	reqBody, _ := json.Marshal(map[string]string{
		"username": "nonexistentuser",
		"password": TestPassword,
	})

	resp, err := http.Post(BaseURL+"/login", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("登录请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证响应状态码
	if resp.StatusCode != 401 {
		t.Errorf("预期状态码 401, 但收到 %d", resp.StatusCode)
	}

	// 解析响应内容
	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证响应内容
	if response.Code != 401 {
		t.Errorf("预期响应代码 401, 但收到 %d", response.Code)
	}
}

// 测试错误的密码
func TestLoginWithInvalidPassword(t *testing.T) {
	// 首先创建一个测试用户
	username := GenerateRandomUsername()
	email := GenerateRandomEmail()

	_, err := RegisterNewUser(username, email, TestPassword)
	if err != nil {
		t.Fatalf("注册测试用户失败: %v", err)
	}

	// 测试登录
	reqBody, _ := json.Marshal(map[string]string{
		"username": username,
		"password": "wrongpassword",
	})

	resp, err := http.Post(BaseURL+"/login", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("登录请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证响应状态码
	if resp.StatusCode != 401 {
		t.Errorf("预期状态码 401, 但收到 %d", resp.StatusCode)
	}

	// 解析响应内容
	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证响应内容
	if response.Code != 401 {
		t.Errorf("预期响应代码 401, 但收到 %d", response.Code)
	}
}

// 测试管理员登录
func TestAdminLogin(t *testing.T) {
	// 从环境变量获取管理员密码
	adminPassword := os.Getenv("DEFAULT_ADMIN_PASSWORD")
	if adminPassword == "" {
		t.Skip("环境变量 DEFAULT_ADMIN_PASSWORD 未设置, 跳过管理员登录测试")
	}

	reqBody, _ := json.Marshal(map[string]string{
		"username": AdminUsername,
		"password": adminPassword,
	})

	resp, err := http.Post(BaseURL+"/login", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("登录请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证响应状态码
	if resp.StatusCode != 200 {
		t.Errorf("预期状态码 200, 但收到 %d", resp.StatusCode)
	}

	// 解析响应内容
	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证响应内容
	if response.Code != 200 {
		t.Errorf("预期响应代码 200, 但收到 %d", response.Code)
	}

	// 验证返回的用户数据和令牌
	var loginData LoginResponseData
	if err := json.Unmarshal(response.Data, &loginData); err != nil {
		t.Fatalf("解析用户数据失败: %v", err)
	}

	if loginData.Token == "" {
		t.Error("预期返回令牌, 但未收到")
	}

	if loginData.User.Username != AdminUsername {
		t.Errorf("预期用户名 %s, 但收到 %s", AdminUsername, loginData.User.Username)
	}

	if loginData.User.Role != "admin" {
		t.Errorf("预期角色 admin, 但收到 %s", loginData.User.Role)
	}
}
