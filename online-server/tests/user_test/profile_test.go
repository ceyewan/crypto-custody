package user_test

import (
	"encoding/json"
	"testing"
)

// 测试获取当前用户信息
func TestGetCurrentUser(t *testing.T) {
	// 首先创建一个测试用户并登录
	username := GenerateRandomUsername()
	email := GenerateRandomEmail()

	_, err := RegisterNewUser(username, email, TestPassword)
	if err != nil {
		t.Fatalf("注册测试用户失败: %v", err)
	}

	token, err := LoginUser(username, TestPassword)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	// 获取当前用户信息
	resp, err := SendAuthenticatedRequest("GET", BaseURL+"/users/profile", token, nil)
	if err != nil {
		t.Fatalf("获取用户信息请求失败: %v", err)
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

	if response.Message != "获取当前用户信息成功" {
		t.Errorf("预期消息 '获取当前用户信息成功', 但收到 '%s'", response.Message)
	}

	// 验证返回的用户数据
	var user UserInfo
	if err := json.Unmarshal(response.Data, &user); err != nil {
		t.Fatalf("解析用户数据失败: %v", err)
	}

	if user.Username != username {
		t.Errorf("预期用户名 %s, 但收到 %s", username, user.Username)
	}

	if user.Email != email {
		t.Errorf("预期邮箱 %s, 但收到 %s", email, user.Email)
	}

	// 验证角色是 guest (默认角色)
	if user.Role != "guest" {
		t.Errorf("预期角色 guest, 但收到 %s", user.Role)
	}
}

// 测试未认证访问
func TestGetCurrentUserUnauthenticated(t *testing.T) {
	// 不提供令牌
	resp, err := SendAuthenticatedRequest("GET", BaseURL+"/users/profile", "", nil)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
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

// 测试修改密码功能
func TestChangePassword(t *testing.T) {
	// 首先创建一个测试用户并登录
	username := GenerateRandomUsername()
	email := GenerateRandomEmail()

	_, err := RegisterNewUser(username, email, TestPassword)
	if err != nil {
		t.Fatalf("注册测试用户失败: %v", err)
	}

	token, err := LoginUser(username, TestPassword)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	// 修改密码
	newPassword := "newpassword456"
	reqBody, _ := json.Marshal(map[string]string{
		"oldPassword": TestPassword,
		"newPassword": newPassword,
	})

	resp, err := SendAuthenticatedRequest("POST", BaseURL+"/users/change-password", token, reqBody)
	if err != nil {
		t.Fatalf("修改密码请求失败: %v", err)
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

	if response.Message != "密码修改成功" {
		t.Errorf("预期消息 '密码修改成功', 但收到 '%s'", response.Message)
	}

	// 测试使用新密码登录
	_, err = LoginUser(username, newPassword)
	if err != nil {
		t.Errorf("使用新密码登录失败: %v", err)
	}

	// 测试使用旧密码登录（应该失败）
	_, err = LoginUser(username, TestPassword)
	if err == nil {
		t.Error("使用旧密码应该登录失败, 但成功了")
	}
}

// 测试使用错误的旧密码修改密码
func TestChangePasswordWithWrongOldPassword(t *testing.T) {
	// 首先创建一个测试用户并登录
	username := GenerateRandomUsername()
	email := GenerateRandomEmail()

	_, err := RegisterNewUser(username, email, TestPassword)
	if err != nil {
		t.Fatalf("注册测试用户失败: %v", err)
	}

	token, err := LoginUser(username, TestPassword)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	// 尝试使用错误的旧密码
	reqBody, _ := json.Marshal(map[string]string{
		"oldPassword": "wrongoldpassword",
		"newPassword": "newpassword456",
	})

	resp, err := SendAuthenticatedRequest("POST", BaseURL+"/users/change-password", token, reqBody)
	if err != nil {
		t.Fatalf("修改密码请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证响应状态码
	if resp.StatusCode != 400 {
		t.Errorf("预期状态码 400, 但收到 %d", resp.StatusCode)
	}

	// 解析响应内容
	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证响应内容
	if response.Code != 400 {
		t.Errorf("预期响应代码 400, 但收到 %d", response.Code)
	}

	// 确认旧密码仍然有效
	_, err = LoginUser(username, TestPassword)
	if err != nil {
		t.Errorf("使用原密码登录应该成功, 但失败了: %v", err)
	}
}
