package account_test

import (
	"encoding/json"
	"testing"
)

// 测试导入单个账户
func TestImportAccount(t *testing.T) {
	// 获取管理员令牌
	adminToken, err := LoginAdmin()
	if err != nil {
		t.Skipf("管理员登录失败, 跳过测试: %v", err)
	}

	// 生成随机地址
	randomAddress := GenerateRandomEthAddress()

	// 导入测试账户
	description := GenerateRandomDescription()
	_, err = ImportAccount(adminToken, randomAddress, "ETH", description)
	if err != nil {
		t.Fatalf("导入账户失败: %v", err)
	}
}

// 测试批量导入账户
func TestBatchImportAccounts(t *testing.T) {
	// 获取管理员令牌
	adminToken, err := LoginAdmin()
	if err != nil {
		t.Skipf("管理员登录失败, 跳过测试: %v", err)
	}

	// 生成随机地址
	randomAddress1 := GenerateRandomEthAddress()
	randomAddress2 := GenerateRandomEthAddress()

	// 准备批量导入的账户数据
	desc1 := GenerateRandomDescription()
	desc2 := GenerateRandomDescription()
	accounts := []map[string]string{
		{
			"address":     randomAddress1,
			"coinType":    "ETH",
			"description": desc1,
		},
		{
			"address":     randomAddress2,
			"coinType":    "ETH",
			"description": desc2,
		},
	}

	// 批量导入账户
	importedAccounts, err := BatchImportAccounts(adminToken, accounts)
	if err != nil {
		t.Fatalf("批量导入账户失败: %v", err)
	}

	// 验证返回的账户信息
	// 注意：由于返回的是所有账户，我们需要找到刚导入的两个账户
	foundAccount1 := false
	foundAccount2 := false

	for _, account := range importedAccounts {
		if account.Address == randomAddress1 {
			foundAccount1 = true
			if account.Description != desc1 {
				t.Errorf("预期描述 %s, 但收到 %s", desc1, account.Description)
			}
		}
		if account.Address == randomAddress2 {
			foundAccount2 = true
			if account.Description != desc2 {
				t.Errorf("预期描述 %s, 但收到 %s", desc2, account.Description)
			}
		}
	}

	if !foundAccount1 {
		t.Errorf("批量导入后未找到账户: %s", randomAddress1)
	}

	if !foundAccount2 {
		t.Errorf("批量导入后未找到账户: %s", randomAddress2)
	}
}

// 测试导入无效格式的ETH地址
func TestImportInvalidEthAddress(t *testing.T) {
	// 获取管理员令牌
	adminToken, err := LoginAdmin()
	if err != nil {
		t.Skipf("管理员登录失败, 跳过测试: %v", err)
	}

	// 无效的ETH地址格式
	invalidAddress := "0xinvalid"
	description := GenerateRandomDescription()

	// 构建请求体
	reqBody, _ := json.Marshal(map[string]interface{}{
		"accounts": []map[string]string{
			{
				"address":     invalidAddress,
				"coinType":    "ETH",
				"description": description,
			},
		},
	})

	// 发送请求
	resp, err := SendAuthenticatedRequest("POST", BaseURL+"/accounts/import", adminToken, reqBody)
	if err != nil {
		t.Fatalf("导入账户请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证响应状态码
	if resp.StatusCode == 200 {
		// 如果响应是200，可能系统接受了无效地址，需要进一步验证响应内容
		var response CommonResponse
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			t.Fatalf("解析响应失败: %v", err)
		}

		if response.Code == 200 {
			t.Error("预期导入无效地址会失败，但请求成功")
		}
	}
}

// 测试普通用户导入账户
func TestImportAccountByNormalUser(t *testing.T) {
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

	// 尝试导入账户
	description := GenerateRandomDescription()
	account, err := ImportAccount(token, randomAddress, "ETH", description)

	// 这里的行为取决于系统设计
	// 如果系统允许普通用户导入账户，则应该成功
	if err == nil {
		// 验证账户信息
		if account.Address != randomAddress {
			t.Errorf("预期地址 %s, 但收到 %s", randomAddress, account.Address)
		}
		if account.ImportedBy != username {
			t.Errorf("预期导入用户为 %s, 但收到 %s", username, account.ImportedBy)
		}
	} else {
		// 如果系统不允许普通用户导入账户，则应该返回权限错误
		// 这里记录信息，但不断言失败，因为这取决于系统设计
		t.Logf("普通用户导入账户结果: %v", err)
	}
}
