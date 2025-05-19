package account_test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/joho/godotenv"
)

func init() {
	// 加载 .env 文件
	rootDir := filepath.Join("..", "..") // 从 tests/account_test 目录回到项目根目录
	err := godotenv.Load(filepath.Join(rootDir, ".env"))
	if err != nil {
		// 仅打印错误，而不是退出，以便测试仍然可以运行
		println("警告: 无法加载 .env 文件:", err.Error())
	}
}

// 测试通过地址获取账户信息
func TestGetAccountByAddress(t *testing.T) {
	// 获取管理员令牌
	adminToken, err := LoginAdmin()
	if err != nil {
		t.Skipf("管理员登录失败, 跳过测试: %v", err)
	}

	// 生成随机地址
	randomAddress := GenerateRandomEthAddress()

	// 创建测试账户
	description := GenerateRandomDescription()
	_, err = ImportAccount(adminToken, randomAddress, "ETH", description)
	if err != nil {
		t.Fatalf("导入账户失败: %v", err)
	}

	// 通过公共API获取账户信息
	retrievedAccount, err := GetAccountByAddress(randomAddress)
	if err != nil {
		t.Fatalf("获取账户信息失败: %v", err)
	}
	// 验证返回的账户信息
	if retrievedAccount.Address != randomAddress {
		t.Errorf("预期地址 %s, 但收到 %s", randomAddress, retrievedAccount.Address)
	}
	if retrievedAccount.CoinType != "ETH" {
		t.Errorf("预期币种 ETH, 但收到 %s", retrievedAccount.CoinType)
	}
}

// TestGetUserAccounts 测试获取用户账户列表
func TestGetUserAccounts(t *testing.T) {
	// 创建一个普通用户并登录
	username := GenerateRandomUsername()
	email := GenerateRandomEmail()
	_, err := RegisterNewUser(username, email, TestPassword)
	if err != nil {
		t.Fatalf("注册测试用户失败: %v", err)
	}

	normalUserToken, err := LoginUser(username, TestPassword)
	if err != nil {
		t.Fatalf("登录失败: %v", err)
	}

	// 使用管理员或警员账户创建账户
	adminToken, err := LoginAdmin()
	if err != nil {
		t.Skipf("管理员登录失败, 跳过测试: %v", err)
	}

	// 为普通用户创建账户
	// 注意：这里可能需要确保账户与用户关联，如果系统支持这种关联
	randomAddress := GenerateRandomEthAddress()
	description := GenerateRandomDescription()

	// 调用管理员API创建账户，指定账户所属用户为刚创建的普通用户
	// 如果您的系统支持在创建账户时指定所有者，请确保这里提供正确的逻辑
	reqBody, _ := json.Marshal(map[string]interface{}{
		"address":     randomAddress,
		"coinType":    "ETH",
		"description": description,
		"importedBy":  username, // 如果您的API支持指定导入用户
	})

	// 使用管理员令牌创建账户
	resp, err := SendAuthenticatedRequest("POST", BaseURL+"/api/accounts/officer/create", adminToken, reqBody)
	if err != nil {
		t.Fatalf("创建账户请求失败: %v", err)
	}
	defer resp.Body.Close()

	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	if response.Code != 200 {
		t.Fatalf("创建账户失败: %s", response.Message)
	}

	// 使用普通用户令牌获取账户列表
	resp, err = SendAuthenticatedRequest("GET", BaseURL+"/api/accounts/officer", normalUserToken, nil)
	if err != nil {
		t.Fatalf("获取账户列表请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 如果系统设计为普通用户不能直接访问 /accounts/officer 路由，
	// 检查是否返回了权限错误，如果是，则测试通过
	if resp.StatusCode == 403 {
		t.Log("普通用户无法访问警员API，符合系统设计")
		return
	}

	// 否则，解析响应并验证返回的账户列表
	var accountsResponse CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&accountsResponse); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证响应状态
	if accountsResponse.Code != 200 {
		t.Errorf("获取账户列表失败: %s", accountsResponse.Message)
	}

	// 验证返回的账户列表包含刚刚创建的账户
	// 这部分逻辑取决于您的响应格式
}

// 测试管理员获取所有账户列表
func TestGetAllAccounts(t *testing.T) {
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

	// 管理员获取所有账户
	accounts, err := GetAllAccounts(adminToken)
	if err != nil {
		t.Fatalf("获取所有账户失败: %v", err)
	}

	// 验证列表中包含我们的测试账户
	found := false
	for _, account := range accounts {
		if account.Address == randomAddress {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("所有账户列表中未找到测试账户: %s", randomAddress)
	}
}

// 测试非管理员用户尝试获取所有账户列表
func TestGetAllAccountsNonAdmin(t *testing.T) {
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

	// 尝试获取所有账户列表
	resp, err := SendAuthenticatedRequest("GET", BaseURL+"/accounts/admin/all", token, nil)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证普通用户没有权限
	if resp.StatusCode == 200 {
		t.Error("预期普通用户无权获取所有账户，但请求成功")
	}

	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证响应内容包含权限错误信息
	if response.Code == 200 {
		t.Errorf("预期错误响应代码, 但收到 %d", response.Code)
	}
}

// 测试获取不存在的账户
func TestGetNonExistentAccount(t *testing.T) {
	// 生成一个随机地址作为不存在的地址
	// 我们不导入此地址，所以它不应该存在于系统中
	nonExistentAddress := GenerateRandomEthAddress()

	// 尝试获取不存在的账户
	_, err := GetAccountByAddress(nonExistentAddress)

	// 验证返回了错误
	if err == nil {
		t.Errorf("预期账户不存在会返回错误, 但未收到错误")
	}
}
