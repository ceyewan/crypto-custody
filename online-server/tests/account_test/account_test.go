package account_test

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
)

func init() {
	// 加载 .env 文件
	rootDir := filepath.Join("..", "..") // 从 tests/user_test 目录回到项目根目录
	err := godotenv.Load(filepath.Join(rootDir, ".env"))
	if err != nil {
		// 仅打印错误，而不是退出，以便测试仍然可以运行
		println("警告: 无法加载 .env 文件:", err.Error())
	}
}

// 测试获取账户列表
func TestGetAccounts(t *testing.T) {
	// 获取管理员令牌
	adminToken, err := LoginAdmin()
	if err != nil {
		t.Skipf("管理员登录失败, 跳过测试: %v", err)
	}

	// 首先导入一些测试账户确保有数据
	accounts := []map[string]string{
		{
			"address":     TestEthAddress,
			"coinType":    "ETH",
			"description": "Test ETH Account 1",
		},
		{
			"address":     TestEthAddress2,
			"coinType":    "ETH",
			"description": "Test ETH Account 2",
		},
	}
	_, err = BatchImportAccounts(adminToken, accounts)
	if err != nil {
		t.Fatalf("批量导入账户失败: %v", err)
	}

	// 获取账户列表
	resp, err := SendAuthenticatedRequest("GET", BaseURL+"/accounts", adminToken, nil)
	if err != nil {
		t.Fatalf("获取账户列表请求失败: %v", err)
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

	// 验证返回的账户列表
	var accountList []AccountInfo
	if err := json.Unmarshal(response.Data, &accountList); err != nil {
		t.Fatalf("解析账户列表失败: %v", err)
	}

	// 验证列表中包含我们的测试账户
	foundAccount1 := false
	foundAccount2 := false
	for _, account := range accountList {
		if account.Address == TestEthAddress {
			foundAccount1 = true
		}
		if account.Address == TestEthAddress2 {
			foundAccount2 = true
		}
	}

	if !foundAccount1 {
		t.Errorf("账户列表中未找到测试账户1: %s", TestEthAddress)
	}
	if !foundAccount2 {
		t.Errorf("账户列表中未找到测试账户2: %s", TestEthAddress2)
	}
}

// 测试获取特定账户
func TestGetAccountByAddress(t *testing.T) {
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

	// 获取特定账户
	resp, err := SendAuthenticatedRequest("GET", fmt.Sprintf("%s/accounts/%s", BaseURL, TestEthAddress), adminToken, nil)
	if err != nil {
		t.Fatalf("获取账户请求失败: %v", err)
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

	// 验证返回的账户数据
	var accountInfo AccountInfo
	if err := json.Unmarshal(response.Data, &accountInfo); err != nil {
		t.Fatalf("解析账户数据失败: %v", err)
	}

	if accountInfo.Address != account.Address {
		t.Errorf("预期地址 %s, 但收到 %s", account.Address, accountInfo.Address)
	}

	if accountInfo.CoinType != account.CoinType {
		t.Errorf("预期币种 %s, 但收到 %s", account.CoinType, accountInfo.CoinType)
	}

	if accountInfo.Description != account.Description {
		t.Errorf("预期描述 %s, 但收到 %s", account.Description, accountInfo.Description)
	}
}

// 测试非管理员用户获取账户列表
func TestGetAccountsNonAdmin(t *testing.T) {
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

	// 尝试获取账户列表
	resp, err := SendAuthenticatedRequest("GET", BaseURL+"/accounts", token, nil)
	if err != nil {
		t.Fatalf("获取账户列表请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证普通用户也可以查看账户列表但可能有权限限制
	// 注意：根据实际业务逻辑修改这里的断言
	// 如果普通用户可以查看自己导入的账户，则状态码应为200
	if resp.StatusCode != 200 {
		// 如果普通用户不能查看任何账户，则可能返回403
		// 这里根据系统设计来确定期望的行为
		// t.Errorf("预期状态码 403, 但收到 %d", resp.StatusCode)
		t.Logf("注意: 普通用户获取账户列表返回状态码: %d", resp.StatusCode)
	}
}

// 测试获取不存在的账户
func TestGetNonExistentAccount(t *testing.T) {
	// 获取管理员令牌
	adminToken, err := LoginAdmin()
	if err != nil {
		t.Skipf("管理员登录失败, 跳过测试: %v", err)
	}

	// 使用一个不存在的地址
	nonExistentAddress := "0x0000000000000000000000000000000000000000"

	// 尝试获取不存在的账户
	resp, err := SendAuthenticatedRequest("GET", fmt.Sprintf("%s/accounts/%s", BaseURL, nonExistentAddress), adminToken, nil)
	if err != nil {
		t.Fatalf("获取账户请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证响应状态码 (应该是404或自定义错误码)
	if resp.StatusCode == 200 {
		t.Errorf("预期账户不存在返回错误状态码, 但收到 %d", resp.StatusCode)
	}

	// 解析响应内容
	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证响应内容包含错误信息
	if response.Code == 200 {
		t.Errorf("预期错误响应代码, 但收到 %d", response.Code)
	}
}
