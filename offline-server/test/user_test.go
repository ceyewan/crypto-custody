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

// 常量定义
const (
	BaseURL         = "http://localhost:8080" // API服务器基础URL
	AdminUsername   = "admin"                 // 管理员用户名
	AdminPassword   = "admin123"              // 管理员密码
	CoordinatorRole = "coordinator"           // 协调者角色
	ParticipantRole = "participant"           // 参与者角色
)

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

// 主测试函数
func RunUserTest() {
	fmt.Println("===== 开始用户测试 =====")

	// 定义用户信息（一个协调者和三个参与者）
	users := []UserInfo{
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

	// 1. 先登录管理员账号
	fmt.Println("1. 登录管理员账号...")
	adminLogin, err := LoginUser(AdminUsername, AdminPassword)
	if err != nil {
		fmt.Printf("管理员登录失败: %v\n", err)
		return
	}
	adminToken := adminLogin.Token
	fmt.Printf("管理员登录成功, 令牌: %s\n", adminToken)

	// 2. 注册所有测试用户
	fmt.Println("\n2. 注册测试用户...")
	for i, user := range users {
		fmt.Printf("注册用户 %d/%d: %s...\n", i+1, len(users), user.Username)
		registerResp, err := RegisterUser(user)
		if err != nil {
			fmt.Printf("注册用户 %s 失败: %v\n", user.Username, err)
			continue
		}
		fmt.Printf("用户 %s 注册成功, 默认角色: %s\n", user.Username, registerResp.User.Role)
	}

	// 3. 使用管理员权限更新用户角色
	fmt.Println("\n3. 更新用户角色...")
	time.Sleep(1 * time.Second) // 简单延迟，确保注册完成

	for i, user := range users {
		fmt.Printf("更新用户 %d/%d: %s 角色为 %s...\n", i+1, len(users), user.Username, user.Role)
		err := UpdateUserRole(adminToken, user.Username, user.Role)
		if err != nil {
			fmt.Printf("更新用户 %s 角色失败: %v\n", user.Username, err)
			continue
		}
		fmt.Printf("用户 %s 角色已更新为 %s\n", user.Username, user.Role)
	}

	// 4. 测试用户登录并获取令牌
	fmt.Println("\n4. 测试用户登录并获取令牌...")
	for i, user := range users {
		fmt.Printf("登录用户 %d/%d: %s...\n", i+1, len(users), user.Username)
		loginResp, err := LoginUser(user.Username, user.Password)
		if err != nil {
			fmt.Printf("登录用户 %s 失败: %v\n", user.Username, err)
			continue
		}
		fmt.Printf("用户 %s 登录成功, 当前角色: %s\n", user.Username, loginResp.User.Role)
		fmt.Printf("令牌: %s\n", loginResp.Token)

		// 更新用户的Token
		users[i].Token = loginResp.Token
	}

	fmt.Println("\n===== 用户测试完成 =====")
}

// TestUserRegisterLoginFlow 使用 Go 的测试框架测试用户注册和登录流程
func TestUserRegisterLoginFlow(t *testing.T) {
	fmt.Println("===== 开始用户测试 =====")

	// 定义用户信息（一个协调者和三个参与者）
	users := []UserInfo{
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

	// 1. 先登录管理员账号
	fmt.Println("1. 登录管理员账号...")
	adminLogin, err := LoginUser(AdminUsername, AdminPassword)
	if err != nil {
		t.Fatalf("管理员登录失败: %v", err)
	}
	adminToken := adminLogin.Token
	fmt.Printf("管理员登录成功, 令牌: %s\n", adminToken)

	// 2. 注册所有测试用户
	fmt.Println("\n2. 注册测试用户...")
	for i, user := range users {
		fmt.Printf("注册用户 %d/%d: %s...\n", i+1, len(users), user.Username)
		registerResp, err := RegisterUser(user)
		if err != nil {
			t.Errorf("注册用户 %s 失败: %v", user.Username, err)
			continue
		}
		fmt.Printf("用户 %s 注册成功, 默认角色: %s\n", user.Username, registerResp.User.Role)
	}

	// 3. 使用管理员权限更新用户角色
	fmt.Println("\n3. 更新用户角色...")
	time.Sleep(1 * time.Second) // 简单延迟，确保注册完成

	for i, user := range users {
		fmt.Printf("更新用户 %d/%d: %s 角色为 %s...\n", i+1, len(users), user.Username, user.Role)
		err := UpdateUserRole(adminToken, user.Username, user.Role)
		if err != nil {
			t.Errorf("更新用户 %s 角色失败: %v", user.Username, err)
			continue
		}
		fmt.Printf("用户 %s 角色已更新为 %s\n", user.Username, user.Role)
	}

	// 4. 测试用户登录并获取令牌
	fmt.Println("\n4. 测试用户登录并获取令牌...")
	for i, user := range users {
		fmt.Printf("登录用户 %d/%d: %s...\n", i+1, len(users), user.Username)
		loginResp, err := LoginUser(user.Username, user.Password)
		if err != nil {
			t.Errorf("登录用户 %s 失败: %v", user.Username, err)
			continue
		}
		fmt.Printf("用户 %s 登录成功, 当前角色: %s\n", user.Username, loginResp.User.Role)
		fmt.Printf("令牌: %s\n", loginResp.Token)

		// 更新用户的Token
		users[i].Token = loginResp.Token
	}

	fmt.Println("\n===== 用户测试完成 =====")
}
