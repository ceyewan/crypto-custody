package transaction_test

import (
	"encoding/json"
	"testing"
)

// 测试准备交易
func TestPrepareTransaction(t *testing.T) {
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

	// 准备交易请求
	prepareReq := PrepareTransactionRequest{
		FromAddress: TestEthAddress,
		ToAddress:   TestEthAddress2,
		Value:       TestValue,
		GasPrice:    TestGasPrice,
		GasLimit:    TestGasLimit,
		Data:        TestDataHex,
		CoinType:    "ETH",
	}

	// 准备交易
	transaction, err := PrepareTransaction(adminToken, prepareReq)
	if err != nil {
		t.Fatalf("准备交易失败: %v", err)
	}

	// 验证交易信息
	if transaction.FromAddress != TestEthAddress {
		t.Errorf("预期发送地址 %s, 但收到 %s", TestEthAddress, transaction.FromAddress)
	}

	if transaction.ToAddress != TestEthAddress2 {
		t.Errorf("预期接收地址 %s, 但收到 %s", TestEthAddress2, transaction.ToAddress)
	}

	if transaction.Value != TestValue {
		t.Errorf("预期金额 %s, 但收到 %s", TestValue, transaction.Value)
	}

	if transaction.GasPrice != TestGasPrice {
		t.Errorf("预期Gas价格 %s, 但收到 %s", TestGasPrice, transaction.GasPrice)
	}

	if transaction.GasLimit != TestGasLimit {
		t.Errorf("预期Gas限制 %s, 但收到 %s", TestGasLimit, transaction.GasLimit)
	}

	if transaction.Status != "prepared" {
		t.Errorf("预期状态 prepared, 但收到 %s", transaction.Status)
	}

	if transaction.ID == "" {
		t.Error("未返回交易ID")
	}

	// 验证生成了原始交易数据
	if transaction.RawTransaction == "" {
		t.Error("未返回原始交易数据")
	}
}

// 测试准备交易使用的是无效的发送地址
func TestPrepareTransactionWithInvalidFromAddress(t *testing.T) {
	// 获取管理员令牌
	adminToken, err := LoginAdmin()
	if err != nil {
		t.Skipf("管理员登录失败, 跳过测试: %v", err)
	}

	// 准备交易请求，使用无效地址
	prepareReq := PrepareTransactionRequest{
		FromAddress: "0xinvalid",
		ToAddress:   TestEthAddress2,
		Value:       TestValue,
		GasPrice:    TestGasPrice,
		GasLimit:    TestGasLimit,
		Data:        TestDataHex,
		CoinType:    "ETH",
	}

	// 构建请求体
	reqBody, _ := json.Marshal(prepareReq)

	// 发送请求
	resp, err := SendAuthenticatedRequest("POST", BaseURL+"/transaction/prepare", adminToken, reqBody)
	if err != nil {
		t.Fatalf("准备交易请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证响应状态码
	if resp.StatusCode == 200 {
		t.Error("预期使用无效地址会失败，但请求成功")
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

// 测试准备交易使用的是无效的接收地址
func TestPrepareTransactionWithInvalidToAddress(t *testing.T) {
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

	// 准备交易请求，使用无效的接收地址
	prepareReq := PrepareTransactionRequest{
		FromAddress: TestEthAddress,
		ToAddress:   "0xinvalid",
		Value:       TestValue,
		GasPrice:    TestGasPrice,
		GasLimit:    TestGasLimit,
		Data:        TestDataHex,
		CoinType:    "ETH",
	}

	// 构建请求体
	reqBody, _ := json.Marshal(prepareReq)

	// 发送请求
	resp, err := SendAuthenticatedRequest("POST", BaseURL+"/transaction/prepare", adminToken, reqBody)
	if err != nil {
		t.Fatalf("准备交易请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证响应状态码
	if resp.StatusCode == 200 {
		t.Error("预期使用无效接收地址会失败，但请求成功")
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

// 测试准备交易使用无效的金额
func TestPrepareTransactionWithInvalidValue(t *testing.T) {
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

	// 准备交易请求，使用无效金额
	prepareReq := PrepareTransactionRequest{
		FromAddress: TestEthAddress,
		ToAddress:   TestEthAddress2,
		Value:       "invalid",
		GasPrice:    TestGasPrice,
		GasLimit:    TestGasLimit,
		Data:        TestDataHex,
		CoinType:    "ETH",
	}

	// 构建请求体
	reqBody, _ := json.Marshal(prepareReq)

	// 发送请求
	resp, err := SendAuthenticatedRequest("POST", BaseURL+"/transaction/prepare", adminToken, reqBody)
	if err != nil {
		t.Fatalf("准备交易请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 验证响应状态码
	if resp.StatusCode == 200 {
		t.Error("预期使用无效金额会失败，但请求成功")
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
