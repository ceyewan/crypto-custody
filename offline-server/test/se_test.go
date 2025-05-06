package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

// 安全芯片测试相关常量
const (
	// 服务器配置
	MpcBaseURL = "http://localhost:8088"

	// 测试用安全芯片ID
	DefaultSEID = "SE000"
)

// 安全芯片CPIC响应结构
type CPICResponse struct {
	Success bool   `json:"success"`
	CPIC    string `json:"cpic"`
}

// 创建安全芯片请求结构
type CreateSeRequest struct {
	SeId string `json:"seId"`
	CPIC string `json:"cpic"`
}

// 创建安全芯片响应结构
type CreateSeResponse struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
}

// GetCPIC 从MPC服务获取CPIC数据
func GetCPIC() (string, error) {
	// 发送GET请求获取CPIC数据
	resp, err := http.Get(fmt.Sprintf("%s/api/v1/mpc/cplc", MpcBaseURL))
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("获取CPIC失败，状态码: %d, 响应: %s", resp.StatusCode, body)
	}

	// 解析响应
	var response CPICResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	if !response.Success {
		return "", fmt.Errorf("获取CPIC未成功")
	}

	return response.CPIC, nil
}

// CreateSecurityElement 创建安全芯片记录
func CreateSecurityElement(token string, seid string, cpic string) error {
	// 构建请求体
	requestBody, err := json.Marshal(CreateSeRequest{
		SeId: seid,
		CPIC: cpic,
	})
	if err != nil {
		return fmt.Errorf("构建请求体失败: %v", err)
	}

	// 创建请求
	req, err := http.NewRequest(
		http.MethodPost,
		fmt.Sprintf("%s/se/create", BaseURL),
		bytes.NewBuffer(requestBody),
	)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", token)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("创建安全芯片记录失败，状态码: %d, 响应: %s", resp.StatusCode, body)
	}

	// 解析响应
	var response CreateSeResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查响应代码
	if response.Code != 0 {
		return fmt.Errorf("创建安全芯片记录失败，代码: %d", response.Code)
	}

	return nil
}

// 完整的安全芯片创建流程，包括获取管理员令牌、获取CPIC、创建安全芯片记录
func CompleteSecurityElementFlow(seid string) error {
	// 1. 获取管理员令牌
	adminToken, err := GetAdminToken()
	if err != nil {
		return fmt.Errorf("获取管理员令牌失败: %v", err)
	}

	// 2. 获取CPIC数据
	cpic, err := GetCPIC()
	if err != nil {
		return fmt.Errorf("获取CPIC数据失败: %v", err)
	}

	// 3. 创建安全芯片记录
	err = CreateSecurityElement(adminToken, seid, cpic)
	if err != nil {
		return fmt.Errorf("创建安全芯片记录失败: %v", err)
	}

	return nil
}

// TestCreateSecurityElement 测试安全芯片记录创建流程
func TestCreateSecurityElement(t *testing.T) {
	fmt.Println("===== 开始安全芯片测试 =====")

	// 使用默认SEID执行安全芯片创建流程
	err := CompleteSecurityElementFlow(DefaultSEID)
	if err != nil {
		t.Fatalf("安全芯片创建失败: %v", err)
	}

	fmt.Println("安全芯片记录创建成功!")
	fmt.Println("\n===== 安全芯片测试完成 =====")
}
