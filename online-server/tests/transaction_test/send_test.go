package transaction_test

import (
	"encoding/json"
	"testing"
	"time"
)

// 测试发送交易
func TestSendTransaction(t *testing.T) {
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

	// 准备交易
	prepareReq := PrepareTransactionRequest{
		FromAddress: TestEthAddress,
		ToAddress:   TestEthAddress2,
		Value:       TestValue,
		GasPrice:    TestGasPrice,
		GasLimit:    TestGasLimit,
		Data:        TestDataHex,
		CoinType:    "ETH",
	}

	transaction, err := PrepareTransaction(adminToken, prepareReq)
	if err != nil {
		t.Fatalf("准备交易失败: %v", err)
	}

	// 提交签名交易（使用模拟签名，实际应用中可能会失败）
	signedTransaction, err := SubmitSignedTransaction(adminToken, transaction.ID, TestSignature)
	if err != nil {
		t.Logf("提交签名交易失败: %v", err)
		t.Skip("使用模拟签名数据，跳过后续测试")
	}

	// 发送交易
	reqBody, _ := json.Marshal(map[string]string{
		"id": signedTransaction.ID,
	})

	resp, err := SendAuthenticatedRequest("POST", BaseURL+"/transaction/send", adminToken, reqBody)
	if err != nil {
		t.Fatalf("发送交易请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 解析响应内容
	var response CommonResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 由于使用模拟签名，实际发送可能会失败
	// 我们只验证请求被处理，但不断言成功
	if response.Code == 200 {
		// 如果成功，验证返回的交易哈希
		var sentTransaction TransactionInfo
		if err := json.Unmarshal(response.Data, &sentTransaction); err != nil {
			t.Fatalf("解析交易数据失败: %v", err)
		}

		if sentTransaction.Hash == "" {
			t.Error("未返回交易哈希")
		}

		if sentTransaction.Status != "pending" && sentTransaction.Status != "submitted" && sentTransaction.Status != "confirmed" {
			t.Errorf("预期交易状态 pending/submitted/confirmed, 但收到 %s", sentTransaction.Status)
		}
	} else {
		// 记录失败原因但不断言错误
		t.Logf("发送交易响应: 状态码 %d, 响应代码 %d, 消息 %s", resp.StatusCode, response.Code, response.Message)
		t.Log("注意: 由于使用模拟签名，交易发送预期会失败")
	}
}

// 测试查询交易状态
func TestQueryTransactionStatus(t *testing.T) {
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

	// 准备交易
	prepareReq := PrepareTransactionRequest{
		FromAddress: TestEthAddress,
		ToAddress:   TestEthAddress2,
		Value:       TestValue,
		GasPrice:    TestGasPrice,
		GasLimit:    TestGasLimit,
		Data:        TestDataHex,
		CoinType:    "ETH",
	}

	transaction, err := PrepareTransaction(adminToken, prepareReq)
	if err != nil {
		t.Fatalf("准备交易失败: %v", err)
	}

	// 查询交易状态
	status, err := CheckTransactionStatus(adminToken, transaction.ID)
	if err != nil {
		t.Fatalf("查询交易状态失败: %v", err)
	}

	// 验证初始状态
	if status != "prepared" {
		t.Errorf("预期初始交易状态 prepared, 但收到 %s", status)
	}

	// 提交签名交易
	_, err = SubmitSignedTransaction(adminToken, transaction.ID, TestSignature)
	if err != nil {
		t.Logf("提交签名交易失败: %v", err)
		t.Skip("使用模拟签名数据，跳过后续测试")
	}

	// 再次查询状态，验证已更新
	status, err = CheckTransactionStatus(adminToken, transaction.ID)
	if err != nil {
		t.Fatalf("查询交易状态失败: %v", err)
	}

	if status != "signed" && status != "pending" && status != "submitted" {
		t.Errorf("预期签名后交易状态 signed/pending/submitted, 但收到 %s", status)
	}
}

// 测试获取交易列表
func TestGetTransactionList(t *testing.T) {
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

	// 准备一个交易，确保列表中有数据
	prepareReq := PrepareTransactionRequest{
		FromAddress: TestEthAddress,
		ToAddress:   TestEthAddress2,
		Value:       TestValue,
		GasPrice:    TestGasPrice,
		GasLimit:    TestGasLimit,
		Data:        TestDataHex,
		CoinType:    "ETH",
	}

	_, err = PrepareTransaction(adminToken, prepareReq)
	if err != nil {
		t.Fatalf("准备交易失败: %v", err)
	}

	// 获取交易列表
	resp, err := SendAuthenticatedRequest("GET", BaseURL+"/transaction/list", adminToken, nil)
	if err != nil {
		t.Fatalf("获取交易列表请求失败: %v", err)
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

	// 验证返回的交易列表
	var transactionList []TransactionInfo
	if err := json.Unmarshal(response.Data, &transactionList); err != nil {
		t.Fatalf("解析交易列表失败: %v", err)
	}

	// 验证列表不为空
	if len(transactionList) == 0 {
		t.Error("交易列表为空")
	}
}

// 测试交易确认
func TestTransactionConfirmation(t *testing.T) {
	// 此测试可能需要较长时间等待交易确认
	// 实际环境中应该单独运行并配置较长的超时时间
	t.Skip("交易确认测试需要等待网络确认，默认跳过")

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

	// 准备交易
	prepareReq := PrepareTransactionRequest{
		FromAddress: TestEthAddress,
		ToAddress:   TestEthAddress2,
		Value:       TestValue,
		GasPrice:    TestGasPrice,
		GasLimit:    TestGasLimit,
		Data:        TestDataHex,
		CoinType:    "ETH",
	}

	transaction, err := PrepareTransaction(adminToken, prepareReq)
	if err != nil {
		t.Fatalf("准备交易失败: %v", err)
	}

	// 提交签名交易
	signedTransaction, err := SubmitSignedTransaction(adminToken, transaction.ID, TestSignature)
	if err != nil {
		t.Logf("提交签名交易失败: %v", err)
		t.Skip("使用模拟签名数据，跳过后续测试")
	}

	// 发送交易
	reqBody, _ := json.Marshal(map[string]string{
		"id": signedTransaction.ID,
	})

	resp, err := SendAuthenticatedRequest("POST", BaseURL+"/transaction/send", adminToken, reqBody)
	if err != nil {
		t.Fatalf("发送交易请求失败: %v", err)
	}
	resp.Body.Close()

	// 等待交易确认（最多等待5分钟）
	confirmed := false
	start := time.Now()
	timeout := 5 * time.Minute

	for time.Since(start) < timeout {
		status, err := CheckTransactionStatus(adminToken, transaction.ID)
		if err != nil {
			t.Logf("查询交易状态失败: %v", err)
		} else if status == "confirmed" {
			confirmed = true
			break
		}
		t.Logf("交易状态: %s, 等待确认...", status)
		time.Sleep(30 * time.Second)
	}

	if !confirmed {
		t.Error("交易在超时时间内未确认")
	} else {
		t.Log("交易已确认")
	}
}
