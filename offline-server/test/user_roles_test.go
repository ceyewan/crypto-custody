package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

const (
	// API端点
	BaseURL       = "http://localhost:8080"
	RegisterURL   = BaseURL + "/user/register"
	LoginURL      = BaseURL + "/user/login"
	UpdateRoleURL = BaseURL + "/user/admin/users/%d/role"
	
	// 用户角色
	RoleCoordinator = "coordinator"
	RoleParticipant = "participant"
)

// 用户注册请求结构
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
}

// 更新角色请求结构
type UpdateRoleRequest struct {
	Role string `json:"role"`
}

// 用户登录请求结构
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// 响应结构
type UserResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

type RegisterResponse struct {
	Message string       `json:"message"`
	User    UserResponse `json:"user"`
}

// 测试主函数
func TestCreateUsersAndSetRoles(t *testing.T) {
	// 第一步：注册四个用户
	users := []RegisterRequest{
		{Username: "coordinator_user", Password: "password123", Email: "coordinator@example.com"},
		{Username: "participant_user1", Password: "password123", Email: "participant1@example.com"},
		{Username: "participant_user2", Password: "password123", Email: "participant2@example.com"},
		{Username: "participant_user3", Password: "password123", Email: "participant3@example.com"},
	}

	userIDs := make([]uint, 4)
	
	// 注册所有用户
	for i, user := range users {
		resp, err := registerUser(user)
		if err != nil {
			t.Fatalf("Failed to register user %s: %v", user.Username, err)
		}
		
		t.Logf("Successfully registered user: %s with ID: %d", user.Username, resp.User.ID)
		userIDs[i] = resp.User.ID
	}
	
	// 第二步：使用admin用户登录
	adminLogin := LoginRequest{
		Username: "admin",
		Password: "admin123", // 默认密码
	}
	
	loginResp, err := loginUser(adminLogin)
	if err != nil {
		t.Fatalf("Failed to login as admin: %v", err)
	}
	
	adminToken := loginResp.Token
	t.Logf("Successfully logged in as admin")
	
	// 第三步：修改用户权限
	// 设置第一个用户为协调者，其他为参与者
	roles := []string{RoleCoordinator, RoleParticipant, RoleParticipant, RoleParticipant}
	
	for i, userID := range userIDs {
		err := updateUserRole(userID, roles[i], adminToken)
		if err != nil {
			t.Fatalf("Failed to update role for user ID %d: %v", userID, err)
		}
		t.Logf("Successfully updated user ID %d to role: %s", userID, roles[i])
	}
	
	t.Log("All users created and roles set successfully")
}

// 注册用户
func registerUser(req RegisterRequest) (*RegisterResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}
	
	resp, err := http.Post(RegisterURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error making register request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	var registerResp RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&registerResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}
	
	return &registerResp, nil
}

// 用户登录
func loginUser(req LoginRequest) (*LoginResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %v", err)
	}
	
	resp, err := http.Post(LoginURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error making login request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	var loginResp LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return nil, fmt.Errorf("error decoding response: %v", err)
	}
	
	return &loginResp, nil
}

// 更新用户角色
func updateUserRole(userID uint, role string, adminToken string) error {
	url := fmt.Sprintf(UpdateRoleURL, userID)
	req := UpdateRoleRequest{Role: role}
	
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshaling request: %v", err)
	}
	
	request, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", adminToken)
	
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("error making update role request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	return nil
}