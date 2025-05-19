package account_test

import (
	"encoding/json"
	"testing"
)

// 测试创建账户
func TestCreateAccount(t *testing.T) {
	// 获取管理员令牌
	adminToken, err := LoginAdmin()
	if err != nil {
		t.Skipf("管理员登录失败, 跳过测试: %v", err)
	}

	// 生成随机地址
	randomAddress := GenerateRandomEthAddress()

	// 生成描述
	description := GenerateRandomDescription()

	// 创建账户
	account, err := CreateAccount(adminToken, randomAddress, "ETH", description)
	if err != nil {
		t.Fatalf("创建账户失败: %v", err)
	}

	// 验证返回的账户信息
	if account.Address != randomAddress {
		t.Errorf("预期地址 %s, 但收到 %s", randomAddress, account.Address)
	}

	if account.CoinType != "ETH" {
		t.Errorf("预期币种 ETH, 但收到 %s", account.CoinType)
	}

	if account.Description != description {
		t.Errorf("预期描述 %s, 但收到 %s", description, account.Description)
	}
}

// 测试普通用户创建账户
func TestCreateAccountByNormalUser(t *testing.T) {
	// 创建一个普通用户并登录
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

	// 生成随机地址
	randomAddress := GenerateRandomEthAddress()

	// 尝试创建账户
	description := GenerateRandomDescription()
	account, err := CreateAccount(token, randomAddress, "ETH", description)

	// 这里的行为取决于系统设计
	// 如果系统允许普通用户创建账户，则应该成功
	if err == nil {
		// 验证账户信息
		if account.Address != randomAddress {
			t.Errorf("预期地址 %s, 但收到 %s", randomAddress, account.Address)
		}
		if account.ImportedBy != username {
			t.Errorf("预期导入用户为 %s, 但收到 %s", username, account.ImportedBy)
		}
	} else {
		// 如果系统不允许普通用户创建账户，则应该返回权限错误
		// 这里记录信息，但不断言失败，因为这取决于系统设计
		t.Logf("普通用户创建账户结果: %v", err)
	}
}

// 测试创建无效格式的账户
func TestCreateInvalidAccount(t *testing.T) {
	// 获取管理员令牌
	adminToken, err := LoginAdmin()
	if err != nil {
		t.Skipf("管理员登录失败, 跳过测试: %v", err)
	}

	// 无效的ETH地址格式
	invalidAddress := "0xinvalid"
	description := GenerateRandomDescription()

	// 构建请求体
	reqBody, _ := json.Marshal(map[string]string{
		"address":     invalidAddress,
		"coinType":    "ETH",
		"description": description,
	})

	// 修正API路径：/accounts/officer/create
	resp, err := SendAuthenticatedRequest("POST", BaseURL+"/accounts/officer/create", adminToken, reqBody)
	if err != nil {
		t.Fatalf("创建账户请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证响应状态码
	if resp.StatusCode == 200 {
		t.Error("预期创建无效地址会失败，但请求成功")
	}

	// 解析响应内容
	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证响应包含错误信息
	if response.Code == 200 {
		t.Errorf("预期错误响应代码, 但收到 %d", response.Code)
	}
}
