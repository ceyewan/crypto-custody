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

// 测试获取用户账户列表
func TestGetUserAccounts(t *testing.T) {
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

	// 导入测试账户
	description := GenerateRandomDescription()
	_, err = ImportAccount(token, randomAddress, "ETH", description)
	if err != nil {
		t.Fatalf("导入账户失败: %v", err)
	}

	// 获取用户的账户列表
	accounts, err := GetAccounts(token)
	if err != nil {
		t.Fatalf("获取账户列表失败: %v", err)
	}

	// 验证列表中包含我们的测试账户
	found := false
	for _, account := range accounts {
		if account.Address == randomAddress {
			found = true
			if account.ImportedBy != username {
				t.Errorf("预期导入用户为 %s, 但收到 %s", username, account.ImportedBy)
			}
			break
		}
	}

	if !found {
		t.Errorf("账户列表中未找到测试账户: %s", randomAddress)
	}
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
