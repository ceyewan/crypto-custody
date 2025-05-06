package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
)

const (
	// 服务器配置
	MpcBaseURL = "http://localhost:8088"
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

// TestCreateSecurityElement 测试安全芯片记录创建流程
func TestCreateSecurityElement(t *testing.T) {
	// 1. 获取管理员令牌
	fmt.Println("1. 登录管理员账号...")
	adminLogin, err := LoginUser(AdminUsername, AdminPassword)
	if err != nil {
		t.Fatalf("管理员登录失败: %v", err)
	}
	adminToken := adminLogin.Token
	fmt.Printf("管理员登录成功，获取令牌: %s\n", adminToken)

	// 2. 获取CPIC数据
	fmt.Println("\n2. 从MPC服务获取CPIC数据...")
	cpic, err := GetCPIC()
	if err != nil {
		t.Fatalf("获取CPIC数据失败: %v", err)
	}
	fmt.Printf("成功获取CPIC数据，长度: %d\n", len(cpic))

	// 3. 提示用户输入SEID
	seid := "SE000"
	fmt.Printf("输入的SEID: %s\n", seid)

	// 4. 创建安全芯片记录
	fmt.Println("\n4. 创建安全芯片记录...")
	err = CreateSecurityElement(adminToken, seid, cpic)
	if err != nil {
		t.Fatalf("创建安全芯片记录失败: %v", err)
	}
	fmt.Println("安全芯片记录创建成功!")
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
	body, err := ioutil.ReadAll(resp.Body)
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
	body, err := ioutil.ReadAll(resp.Body)
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
