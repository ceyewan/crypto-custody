package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

// 全局常量定义
const (
	// 服务器配置
	BaseURL = "http://localhost:8080" // API服务器基础URL

	// 用户角色常量
	AdminUsername   = "admin"       // 管理员用户名
	AdminPassword   = "admin123"    // 管理员密码
	CoordinatorRole = "coordinator" // 协调者角色
	ParticipantRole = "participant" // 参与者角色
)

// 全局测试用户数据
var TestUsers = []UserInfo{
	{
		Username: "coordinator",
		Password: "password123",
		Email:    "coordinator@example.com",
		Role:     CoordinatorRole,
	},
	{
		Username: "participant1",
		Password: "password123",
		Email:    "participant1@example.com",
		Role:     ParticipantRole,
	},
	{
		Username: "participant2",
		Password: "password123",
		Email:    "participant2@example.com",
		Role:     ParticipantRole,
	},
	{
		Username: "participant3",
		Password: "password123",
		Email:    "participant3@example.com",
		Role:     ParticipantRole,
	},
}

// 用户信息结构体
type UserInfo struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Token    string `json:"token"`
}

// 登录响应结构体
type LoginResponse struct {
	Token string `json:"token"`
	User  struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	} `json:"user"`
}

// 注册响应结构体
type RegisterResponse struct {
	Message string `json:"message"`
	User    struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	} `json:"user"`
}

// UpdateRoleResponse 角色更新响应结构体
type UpdateRoleResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// 注册用户
func RegisterUser(user UserInfo) (*RegisterResponse, error) {
	// 构建请求体
	requestBody, err := json.Marshal(map[string]string{
		"username": user.Username,
		"password": user.Password,
		"email":    user.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("构建请求体失败: %v", err)
	}

	// 发送注册请求
	resp, err := http.Post(
		fmt.Sprintf("%s/user/register", BaseURL),
		"application/json",
		bytes.NewBuffer(requestBody),
	)
	if err != nil {
		return nil, fmt.Errorf("发送注册请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("注册失败: %s, 状态码: %d", body, resp.StatusCode)
	}

	// 解析响应
	var response RegisterResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return &response, nil
}

// 用户登录
func LoginUser(username, password string) (*LoginResponse, error) {
	// 构建请求体
	requestBody, err := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})
	if err != nil {
		return nil, fmt.Errorf("构建请求体失败: %v", err)
	}

	// 发送登录请求
	resp, err := http.Post(
		fmt.Sprintf("%s/user/login", BaseURL),
		"application/json",
		bytes.NewBuffer(requestBody),
	)
	if err != nil {
		return nil, fmt.Errorf("发送登录请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("登录失败: %s, 状态码: %d", body, resp.StatusCode)
	}

	// 解析响应
	var response LoginResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return &response, nil
}

// 更新用户角色
func UpdateUserRole(token string, username, role string) error {
	// 构建请求体
	requestBody, err := json.Marshal(map[string]string{
		"role": role,
	})
	if err != nil {
		return fmt.Errorf("构建请求体失败: %v", err)
	}

	// 创建PUT请求
	req, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("%s/user/admin/users/%s/role", BaseURL, username),
		bytes.NewBuffer(requestBody),
	)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("更新角色失败: %s, 状态码: %d", body, resp.StatusCode)
	}

	// 解析响应
	var response UpdateRoleResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查响应代码
	if response.Code != 200 {
		return fmt.Errorf("更新角色失败: %s", response.Msg)
	}

	return nil
}

// 管理员账户登录并返回令牌
func GetAdminToken() (string, error) {
	adminLogin, err := LoginUser(AdminUsername, AdminPassword)
	if err != nil {
		return "", fmt.Errorf("管理员登录失败: %v", err)
	}
	return adminLogin.Token, nil
}

// 注册并设置用户角色
func RegisterAndSetupUsers() error {
	// 1. 获取管理员令牌
	adminToken, err := GetAdminToken()
	if err != nil {
		return fmt.Errorf("获取管理员令牌失败: %v", err)
	}

	// 2. 注册所有测试用户
	for _, user := range TestUsers {
		_, err := RegisterUser(user)
		if err != nil {
			return fmt.Errorf("注册用户 %s 失败: %v", user.Username, err)
		}
	}

	// 3. 允许用户注册完成
	time.Sleep(1 * time.Second)

	// 4. 为所有用户设置正确的角色
	for _, user := range TestUsers {
		err := UpdateUserRole(adminToken, user.Username, user.Role)
		if err != nil {
			return fmt.Errorf("更新用户 %s 角色失败: %v", user.Username, err)
		}
	}

	return nil
}

// 登录所有测试用户
func LoginAllUsers() error {
	for i, user := range TestUsers {
		loginResp, err := LoginUser(user.Username, user.Password)
		if err != nil {
			return fmt.Errorf("登录用户 %s 失败: %v", user.Username, err)
		}

		// 更新全局用户数据中的令牌
		TestUsers[i].Token = loginResp.Token
	}
	return nil
}

// TestUserRegisterLoginFlow 使用 Go 的测试框架测试用户注册和登录流程
func TestUserRegisterLoginFlow(t *testing.T) {
	fmt.Println("===== 开始用户测试 =====")

	// 注册并设置用户角色
	err := RegisterAndSetupUsers()
	if err != nil {
		t.Fatalf("用户注册与设置失败: %v", err)
	}

	// 登录所有测试用户
	err = LoginAllUsers()
	if err != nil {
		t.Fatalf("用户登录失败: %v", err)
	}

	fmt.Println("用户注册和登录成功完成")
	fmt.Println("\n===== 用户测试完成 =====")
}
