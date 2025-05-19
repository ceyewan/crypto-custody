package transaction_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// 测试ETH余额查询
func TestGetBalance(t *testing.T) {
	// 准备请求
	url := fmt.Sprintf("%s/transaction/balance/%s", BaseURL, TestAccountB)

	// 发送请求
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("获取余额请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("读取响应体失败: %v", err)
	}

	// 解析响应
	var response CommonResponse
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("解析响应失败: %v", err)
	}

	// 验证响应状态码
	assert.Equal(t, 200, response.Code, "余额查询应该成功")

	// 解析余额数据
	var balanceData BalanceResponse
	if err := json.Unmarshal(response.Data, &balanceData); err != nil {
		t.Fatalf("解析余额数据失败: %v", err)
	}

	// 验证返回的地址与请求的一致
	assert.Equal(t, TestAccountB, balanceData.Address, "返回的地址应该与请求的地址一致")

	// 验证余额不为空
	assert.NotEmpty(t, balanceData.Balance, "余额不应该为空")

	t.Logf("地址 %s 的余额: %s", balanceData.Address, balanceData.Balance)
}

// 测试完整的转账流程
func TestEthTransfer(t *testing.T) {
	// 检查测试账户地址是否设置
	if TestAccountA == "0x" || TestAccountB == "0x" {
		t.Fatal("请先在common_test.go中设置测试账户地址")
	}

	// 检查签名是否设置
	if TestSignature == "" {
		t.Fatal("请先在common_test.go中设置测试交易签名")
	}

	// 1. 登录警员账号
	token, err := LoginOfficer()
	if err != nil {
		t.Fatalf("警员登录失败: %v", err)
	}
	assert.NotEmpty(t, token, "登录应该返回token")

	// 2. 准备交易
	messageHash, err := prepareTransaction(t, token)
	if err != nil {
		t.Fatalf("准备交易失败: %v", err)
	}
	assert.NotEmpty(t, messageHash, "准备交易应返回消息哈希")

	// 3. 签名并发送交易
	txHash, err := signAndSendTransaction(t, token, messageHash)
	if err != nil {
		t.Fatalf("签名并发送交易失败: %v", err)
	}
	assert.NotEmpty(t, txHash, "交易发送应返回交易哈希")

	// 4. 等待交易确认 (通常需要几个区块确认)
	t.Log("等待交易确认...")
	time.Sleep(15 * time.Second)

	// 5. 检查接收方余额
	t.Log("检查接收方余额...")
	TestGetBalance(t)
}

// 准备交易
func prepareTransaction(t *testing.T, token string) (string, error) {
	// 准备请求体
	reqData := map[string]interface{}{
		"fromAddress": TestAccountA,
		"toAddress":   TestAccountB,
		"amount":      TestAmount,
	}
	reqBody, err := json.Marshal(reqData)
	if err != nil {
		return "", err
	}

	// 创建请求
	req, err := http.NewRequest("POST", BaseURL+"/transaction/tx/prepare", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 解析响应
	var response CommonResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", err
	}

	// 检查状态码
	if response.Code != 200 {
		return "", fmt.Errorf("API错误: %d - %s", response.Code, response.Message)
	}

	// 解析交易准备数据
	var prepareData PrepareTransactionResponse
	if err := json.Unmarshal(response.Data, &prepareData); err != nil {
		return "", err
	}

	t.Logf("交易准备成功，交易ID: %d, 消息哈希: %s", prepareData.TransactionID, prepareData.MessageHash)
	return prepareData.MessageHash, nil
}

// 签名并发送交易
func signAndSendTransaction(t *testing.T, token string, messageHash string) (string, error) {
	// 准备请求体
	reqData := map[string]string{
		"messageHash": messageHash,
		"signature":   TestSignature,
	}
	reqBody, err := json.Marshal(reqData)
	if err != nil {
		return "", err
	}

	// 创建请求
	req, err := http.NewRequest("POST", BaseURL+"/transaction/tx/sign-send", bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 解析响应
	var response CommonResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", err
	}

	// 检查状态码
	if response.Code != 200 {
		return "", fmt.Errorf("API错误: %d - %s", response.Code, response.Message)
	}

	// 解析交易发送数据
	var sendData SendTransactionResponse
	if err := json.Unmarshal(response.Data, &sendData); err != nil {
		return "", err
	}

	t.Logf("交易发送成功，交易ID: %d, 交易哈希: %s", sendData.TransactionID, sendData.TxHash)
	return sendData.TxHash, nil
}
