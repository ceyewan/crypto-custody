package user_test

import (
	"encoding/json"
	"fmt"
	"testing"
)

// 测试获取所有用户
func TestGetAllUsers(t *testing.T) {
	// 获取管理员令牌
	adminToken, err := LoginAdmin()
	if err != nil {
		t.Skipf("管理员登录失败, 跳过测试: %v", err)
	}

	// 获取所有用户
	resp, err := SendAuthenticatedRequest("GET", BaseURL+"/users/admin/users", adminToken, nil)
	if err != nil {
		t.Fatalf("获取用户列表请求失败: %v", err)
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

	if response.Message != "获取用户列表成功" {
		t.Errorf("预期消息 '获取用户列表成功', 但收到 '%s'", response.Message)
	}

	// 验证返回的用户列表
	var users []UserInfo
	if err := json.Unmarshal(response.Data, &users); err != nil {
		t.Fatalf("解析用户列表失败: %v", err)
	}

	// 确认至少有一个管理员用户
	hasAdmin := false
	for _, user := range users {
		if user.Role == "admin" {
			hasAdmin = true
			break
		}
	}
	if !hasAdmin {
		t.Error("用户列表中应该至少有一个管理员用户")
	}
}

// 测试非管理员尝试获取所有用户
func TestGetAllUsersNonAdmin(t *testing.T) {
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

	// 尝试获取所有用户
	resp, err := SendAuthenticatedRequest("GET", BaseURL+"/users/admin/users", token, nil)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证响应状态码
	if resp.StatusCode != 403 {
		t.Errorf("预期状态码 403, 但收到 %d", resp.StatusCode)
	}

	// 解析响应内容
	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证响应内容
	if response.Code != 403 {
		t.Errorf("预期响应代码 403, 但收到 %d", response.Code)
	}
}

// 测试获取特定用户
func TestGetSpecificUser(t *testing.T) {
	// 获取管理员令牌
	adminToken, err := LoginAdmin()
	if err != nil {
		t.Skipf("管理员登录失败, 跳过测试: %v", err)
	}

	// 首先创建一个测试用户
	username := GenerateRandomUsername()
	email := GenerateRandomEmail()

	user, err := RegisterNewUser(username, email, TestPassword)
	if err != nil {
		t.Fatalf("注册测试用户失败: %v", err)
	}

	// 获取该用户
	resp, err := SendAuthenticatedRequest("GET", fmt.Sprintf("%s/users/admin/users/%d", BaseURL, user.ID), adminToken, nil)
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

	// 验证返回的用户数据
	var userInfo UserInfo
	if err := json.Unmarshal(response.Data, &userInfo); err != nil {
		t.Fatalf("解析用户数据失败: %v", err)
	}

	if userInfo.Username != username {
		t.Errorf("预期用户名 %s, 但收到 %s", username, userInfo.Username)
	}

	if userInfo.Email != email {
		t.Errorf("预期邮箱 %s, 但收到 %s", email, userInfo.Email)
	}
}

// 测试更改用户角色
func TestChangeUserRole(t *testing.T) {
	// 获取管理员令牌
	adminToken, err := LoginAdmin()
	if err != nil {
		t.Skipf("管理员登录失败, 跳过测试: %v", err)
	}

	// 首先创建一个测试用户
	username := GenerateRandomUsername()
	email := GenerateRandomEmail()

	user, err := RegisterNewUser(username, email, TestPassword)
	if err != nil {
		t.Fatalf("注册测试用户失败: %v", err)
	}

	// 将用户角色更改为 officer
	reqBody, _ := json.Marshal(map[string]string{
		"role": "officer",
	})

	resp, err := SendAuthenticatedRequest("PUT", fmt.Sprintf("%s/users/admin/users/%d/role", BaseURL, user.ID), adminToken, reqBody)
	if err != nil {
		t.Fatalf("更改用户角色请求失败: %v", err)
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

	// 获取该用户确认角色已更改
	resp, err = SendAuthenticatedRequest("GET", fmt.Sprintf("%s/users/admin/users/%d", BaseURL, user.ID), adminToken, nil)
	if err != nil {
		t.Fatalf("获取用户信息请求失败: %v", err)
	}
	defer resp.Body.Close()

	var userResponse CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResponse); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	var userInfo UserInfo
	if err := json.Unmarshal(userResponse.Data, &userInfo); err != nil {
		t.Fatalf("解析用户数据失败: %v", err)
	}

	if userInfo.Role != "officer" {
		t.Errorf("预期角色已更改为 officer, 但收到 %s", userInfo.Role)
	}
}

// 测试更改用户名
func TestChangeUsername(t *testing.T) {
	// 获取管理员令牌
	adminToken, err := LoginAdmin()
	if err != nil {
		t.Skipf("管理员登录失败, 跳过测试: %v", err)
	}

	// 首先创建一个测试用户
	username := GenerateRandomUsername()
	email := GenerateRandomEmail()

	user, err := RegisterNewUser(username, email, TestPassword)
	if err != nil {
		t.Fatalf("注册测试用户失败: %v", err)
	}

	// 更改用户名
	newUsername := GenerateRandomUsername()
	reqBody, _ := json.Marshal(map[string]string{
		"username": newUsername,
	})

	resp, err := SendAuthenticatedRequest("PUT", fmt.Sprintf("%s/users/admin/users/%d/username", BaseURL, user.ID), adminToken, reqBody)
	if err != nil {
		t.Fatalf("更改用户名请求失败: %v", err)
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

	// 获取该用户确认用户名已更改
	resp, err = SendAuthenticatedRequest("GET", fmt.Sprintf("%s/users/admin/users/%d", BaseURL, user.ID), adminToken, nil)
	if err != nil {
		t.Fatalf("获取用户信息请求失败: %v", err)
	}
	defer resp.Body.Close()

	var userResponse CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&userResponse); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	var userInfo UserInfo
	if err := json.Unmarshal(userResponse.Data, &userInfo); err != nil {
		t.Fatalf("解析用户数据失败: %v", err)
	}

	if userInfo.Username != newUsername {
		t.Errorf("预期用户名已更改为 %s, 但收到 %s", newUsername, userInfo.Username)
	}

	// 确认用户仍可以使用新用户名登录
	_, err = LoginUser(newUsername, TestPassword)
	if err != nil {
		t.Errorf("使用新用户名登录失败: %v", err)
	}
}

// 测试删除用户
func TestDeleteUser(t *testing.T) {
	// 获取管理员令牌
	adminToken, err := LoginAdmin()
	if err != nil {
		t.Skipf("管理员登录失败, 跳过测试: %v", err)
	}

	// 首先创建一个测试用户
	username := GenerateRandomUsername()
	email := GenerateRandomEmail()

	user, err := RegisterNewUser(username, email, TestPassword)
	if err != nil {
		t.Fatalf("注册测试用户失败: %v", err)
	}

	// 删除用户
	resp, err := SendAuthenticatedRequest("DELETE", fmt.Sprintf("%s/users/admin/users/%d", BaseURL, user.ID), adminToken, nil)
	if err != nil {
		t.Fatalf("删除用户请求失败: %v", err)
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

	// 尝试获取被删除的用户，应该返回404
	resp, err = SendAuthenticatedRequest("GET", fmt.Sprintf("%s/users/admin/users/%d", BaseURL, user.ID), adminToken, nil)
	if err != nil {
		t.Fatalf("获取用户信息请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("删除用户后，预期获取用户返回状态码 404, 但收到 %d", resp.StatusCode)
	}

	// 尝试使用被删除用户的凭据登录，应该失败
	_, err = LoginUser(username, TestPassword)
	if err == nil {
		t.Error("预期使用已删除用户的凭据登录应该失败，但成功了")
	}
}
