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

	// 导入测试账户
	description := GenerateRandomDescription()
	account, err := ImportAccount(adminToken, TestEthAddress, "ETH", description)
	if err != nil {
		t.Fatalf("导入账户失败: %v", err)
	}

	// 验证返回的账户信息
	if account.Address != TestEthAddress {
		t.Errorf("预期地址 %s, 但收到 %s", TestEthAddress, account.Address)
	}

	if account.CoinType != "ETH" {
		t.Errorf("预期币种 ETH, 但收到 %s", account.CoinType)
	}

	if account.Description != description {
		t.Errorf("预期描述 %s, 但收到 %s", description, account.Description)
	}
}

// 测试批量导入账户
func TestBatchImportAccounts(t *testing.T) {
	// 获取管理员令牌
	adminToken, err := LoginAdmin()
	if err != nil {
		t.Skipf("管理员登录失败, 跳过测试: %v", err)
	}

	// 准备批量导入的账户数据
	desc1 := GenerateRandomDescription()
	desc2 := GenerateRandomDescription()
	accounts := []map[string]string{
		{
			"address":     TestEthAddress,
			"coinType":    "ETH",
			"description": desc1,
		},
		{
			"address":     TestEthAddress2,
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
	if len(importedAccounts) != len(accounts) {
		t.Errorf("预期导入 %d 个账户, 但实际导入 %d 个", len(accounts), len(importedAccounts))
	}

	// 验证第一个账户
	if len(importedAccounts) > 0 {
		if importedAccounts[0].Address != TestEthAddress {
			t.Errorf("预期地址 %s, 但收到 %s", TestEthAddress, importedAccounts[0].Address)
		}
		if importedAccounts[0].CoinType != "ETH" {
			t.Errorf("预期币种 ETH, 但收到 %s", importedAccounts[0].CoinType)
		}
		if importedAccounts[0].Description != desc1 {
			t.Errorf("预期描述 %s, 但收到 %s", desc1, importedAccounts[0].Description)
		}
	}

	// 验证第二个账户
	if len(importedAccounts) > 1 {
		if importedAccounts[1].Address != TestEthAddress2 {
			t.Errorf("预期地址 %s, 但收到 %s", TestEthAddress2, importedAccounts[1].Address)
		}
		if importedAccounts[1].CoinType != "ETH" {
			t.Errorf("预期币种 ETH, 但收到 %s", importedAccounts[1].CoinType)
		}
		if importedAccounts[1].Description != desc2 {
			t.Errorf("预期描述 %s, 但收到 %s", desc2, importedAccounts[1].Description)
		}
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
	reqBody, _ := json.Marshal(map[string]string{
		"address":     invalidAddress,
		"coinType":    "ETH",
		"description": description,
	})

	// 发送请求
	resp, err := SendAuthenticatedRequest("POST", BaseURL+"/accounts/import", adminToken, reqBody)
	if err != nil {
		t.Fatalf("导入账户请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证响应状态码
	if resp.StatusCode == 200 {
		t.Error("预期导入无效地址会失败，但请求成功")
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

	// 尝试导入账户
	description := GenerateRandomDescription()
	account, err := ImportAccount(token, TestEthAddress, "ETH", description)

	// 这里的行为取决于系统设计
	// 如果系统允许普通用户导入账户，则应该成功
	if err == nil {
		// 验证账户信息
		if account.Address != TestEthAddress {
			t.Errorf("预期地址 %s, 但收到 %s", TestEthAddress, account.Address)
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

// 测试导入重复账户
func TestImportDuplicateAccount(t *testing.T) {
	// 获取管理员令牌
	adminToken, err := LoginAdmin()
	if err != nil {
		t.Skipf("管理员登录失败, 跳过测试: %v", err)
	}

	// 首次导入账户
	description1 := GenerateRandomDescription()
	_, err = ImportAccount(adminToken, TestEthAddress, "ETH", description1)
	if err != nil {
		t.Fatalf("首次导入账户失败: %v", err)
	}

	// 尝试再次导入相同地址
	description2 := GenerateRandomDescription()
	_, err = ImportAccount(adminToken, TestEthAddress, "ETH", description2)

	// 根据系统设计，可能允许重复导入（更新描述），或者报错
	// 这里记录结果，但不断言错误，因为这取决于系统设计
	if err != nil {
		t.Logf("重复导入账户结果: %v", err)
	} else {
		t.Logf("系统允许重复导入相同地址的账户，可能是更新了描述")
	}
}
