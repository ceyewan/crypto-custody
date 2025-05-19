package user_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

// 测试成功注册
func TestRegisterSuccess(t *testing.T) {
	username := GenerateRandomUsername()
	email := GenerateRandomEmail()

	reqBody, _ := json.Marshal(map[string]string{
		"username": username,
		"password": TestPassword,
		"email":    email,
	})

	resp, err := http.Post(BaseURL+"/register", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("注册请求失败: %v", err)
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

	if response.Message != "注册成功" {
		t.Errorf("预期消息 '注册成功', 但收到 '%s'", response.Message)
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
}

// 测试用户名已存在
func TestRegisterDuplicateUsername(t *testing.T) {
	// 首先创建一个测试用户
	username := GenerateRandomUsername()
	email := GenerateRandomEmail()

	_, err := RegisterNewUser(username, email, TestPassword)
	if err != nil {
		t.Fatalf("注册测试用户失败: %v", err)
	}

	// 尝试用相同的用户名再次注册
	reqBody, _ := json.Marshal(map[string]string{
		"username": username,
		"password": TestPassword,
		"email":    GenerateRandomEmail(), // 使用不同的邮箱
	})

	resp, err := http.Post(BaseURL+"/register", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("注册请求失败: %v", err)
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

	if response.Message == "" || response.Message == "注册成功" {
		t.Errorf("预期错误消息关于用户名已存在, 但收到 '%s'", response.Message)
	}
}

// 测试邮箱已存在
func TestRegisterDuplicateEmail(t *testing.T) {
	// 首先创建一个测试用户
	username := GenerateRandomUsername()
	email := GenerateRandomEmail()

	_, err := RegisterNewUser(username, email, TestPassword)
	if err != nil {
		t.Fatalf("注册测试用户失败: %v", err)
	}

	// 尝试用相同的邮箱再次注册
	reqBody, _ := json.Marshal(map[string]string{
		"username": GenerateRandomUsername(), // 使用不同的用户名
		"password": TestPassword,
		"email":    email, // 使用相同的邮箱
	})

	resp, err := http.Post(BaseURL+"/register", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("注册请求失败: %v", err)
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

	if response.Message == "" || response.Message == "注册成功" {
		t.Errorf("预期错误消息关于邮箱已被使用, 但收到 '%s'", response.Message)
	}
}

// 测试无效的注册数据
func TestRegisterInvalidData(t *testing.T) {
	// 缺少邮箱
	t.Run("MissingEmail", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]string{
			"username": GenerateRandomUsername(),
			"password": TestPassword,
			// 缺少email字段
		})

		resp, err := http.Post(BaseURL+"/register", "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatalf("注册请求失败: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 400 {
			t.Errorf("预期状态码 400, 但收到 %d", resp.StatusCode)
		}
	})

	// 缺少密码
	t.Run("MissingPassword", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]string{
			"username": GenerateRandomUsername(),
			"email":    GenerateRandomEmail(),
			// 缺少password字段
		})

		resp, err := http.Post(BaseURL+"/register", "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatalf("注册请求失败: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 400 {
			t.Errorf("预期状态码 400, 但收到 %d", resp.StatusCode)
		}
	})

	// 无效的邮箱格式
	t.Run("InvalidEmailFormat", func(t *testing.T) {
		reqBody, _ := json.Marshal(map[string]string{
			"username": GenerateRandomUsername(),
			"password": TestPassword,
			"email":    "notavalidemail", // 无效的邮箱格式
		})

		resp, err := http.Post(BaseURL+"/register", "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			t.Fatalf("注册请求失败: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 400 {
			t.Errorf("预期状态码 400, 但收到 %d", resp.StatusCode)
		}
	})
}
