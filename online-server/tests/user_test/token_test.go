package user_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

// 测试有效令牌验证
func TestCheckValidToken(t *testing.T) {
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

	// 验证令牌
	reqBody, _ := json.Marshal(map[string]string{
		"token": token,
	})

	resp, err := http.Post(BaseURL+"/check-auth", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("验证令牌请求失败: %v", err)
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

	if response.Message != "令牌有效" {
		t.Errorf("预期消息 '令牌有效', 但收到 '%s'", response.Message)
	}

	if response.Valid == nil || *response.Valid != true {
		t.Error("预期 valid=true, 但未收到或值不是true")
	}

	// 验证返回的用户数据
	var data struct {
		User UserInfo `json:"user"`
	}
	if err := json.Unmarshal(response.Data, &data); err != nil {
		t.Fatalf("解析用户数据失败: %v", err)
	}

	if data.User.Username != username {
		t.Errorf("预期用户名 %s, 但收到 %s", username, data.User.Username)
	}
}

// 测试无效令牌
func TestCheckInvalidToken(t *testing.T) {
	// 使用明显无效的令牌
	invalidToken := "invalid.token.string"

	reqBody, _ := json.Marshal(map[string]string{
		"token": invalidToken,
	})

	resp, err := http.Post(BaseURL+"/check-auth", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("验证令牌请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 解析响应内容
	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证响应内容
	if response.Code != 401 {
		t.Errorf("预期响应代码 401, 但收到 %d", response.Code)
	}

	if response.Valid == nil || *response.Valid != false {
		t.Error("预期 valid=false, 但未收到或值不是false")
	}
}

// 测试登出后的令牌
func TestCheckTokenAfterLogout(t *testing.T) {
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

	// 执行登出
	_, err = SendAuthenticatedRequest("POST", BaseURL+"/users/logout", token, nil)
	if err != nil {
		t.Fatalf("登出请求失败: %v", err)
	}

	// 验证已登出的令牌
	reqBody, _ := json.Marshal(map[string]string{
		"token": token,
	})

	resp, err := http.Post(BaseURL+"/check-auth", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("验证令牌请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 解析响应内容
	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证响应内容 - 预期令牌无效
	if response.Code != 401 {
		t.Errorf("预期响应代码 401, 但收到 %d", response.Code)
	}

	if response.Valid == nil || *response.Valid != false {
		t.Error("预期 valid=false, 但未收到或值不是false")
	}
}
