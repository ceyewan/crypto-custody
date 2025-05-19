package account_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"
)

// 测试获取账户余额
func TestGetAccountBalance(t *testing.T) {
	// 获取管理员令牌
	adminToken, err := LoginAdmin()
	if err != nil {
		t.Skipf("管理员登录失败, 跳过测试: %v", err)
	}

	// 导入测试账户
	description := GenerateRandomDescription()
	_, err = ImportAccount(adminToken, TestEthAddress, "ETH", description)
	if err != nil {
		t.Fatalf("导入账户失败: %v", err)
	}

	// 获取账户余额
	balance, err := GetAccountBalance(TestEthAddress)
	if err != nil {
		t.Fatalf("获取账户余额失败: %v", err)
	}

	// 验证余额是否为有效值
	_, err = strconv.ParseFloat(balance, 64)
	if err != nil {
		t.Errorf("返回的余额 '%s' 不是有效的数值", balance)
	}

	t.Logf("账户 %s 的余额为 %s ETH", TestEthAddress, balance)
}

// 测试获取无效账户的余额
func TestGetInvalidAccountBalance(t *testing.T) {
	// 使用一个无效或不存在的地址
	invalidAddress := "0xinvalid"

	// 直接发送请求，不使用辅助函数以获取更详细的响应
	resp, err := http.Get(fmt.Sprintf("%s/transaction/balance/%s", BaseURL, invalidAddress))
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证响应状态码
	if resp.StatusCode == 200 {
		t.Error("预期获取无效地址余额会失败，但请求成功")
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

// 测试批量获取账户余额
func TestBatchGetAccountBalances(t *testing.T) {
	// 获取管理员令牌
	adminToken, err := LoginAdmin()
	if err != nil {
		t.Skipf("管理员登录失败, 跳过测试: %v", err)
	}

	// 导入多个测试账户
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

	// 构建获取批量余额的请求体
	addresses := []string{TestEthAddress, TestEthAddress2}
	reqBody, _ := json.Marshal(map[string]interface{}{
		"addresses": addresses,
	})

	// 发送获取批量余额的请求
	resp, err := SendAuthenticatedRequest("POST", BaseURL+"/transaction/balances", adminToken, reqBody)
	if err != nil {
		t.Fatalf("获取批量余额请求失败: %v", err)
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

	// 验证返回的余额信息
	var balances []struct {
		Address string `json:"address"`
		Balance string `json:"balance"`
	}
	if err := json.Unmarshal(response.Data, &balances); err != nil {
		t.Fatalf("解析余额数据失败: %v", err)
	}

	// 验证返回了所有请求的地址的余额
	if len(balances) != len(addresses) {
		t.Errorf("预期返回 %d 个余额记录, 但实际返回 %d 个", len(addresses), len(balances))
	}

	// 验证每个地址的余额格式正确
	for _, balanceInfo := range balances {
		found := false
		for _, addr := range addresses {
			if balanceInfo.Address == addr {
				found = true
				// 验证余额是否为有效值
				_, err = strconv.ParseFloat(balanceInfo.Balance, 64)
				if err != nil {
					t.Errorf("地址 %s 的余额 '%s' 不是有效的数值", balanceInfo.Address, balanceInfo.Balance)
				}
				break
			}
		}
		if !found {
			t.Errorf("返回了未请求的地址 %s 的余额", balanceInfo.Address)
		}
	}
}
