package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
)

// 签名相关常量
const (
	// 签名数据
	TestSignData = "0x1234abcd5678efgh9012ijkl3456mnop7890qrst"
	// 签名地址
	address = "0x6A546C2CA174dac0f9D9ac91D959d97597c9A06b"
)

// 创建签名会话密钥响应结构体
type CreateSignSessionResponse struct {
	SessionKey string `json:"session_key"`
}

// 签名请求结构体
type SignRequest struct {
	Parties      string `json:"parties"`      // 参与者索引列表，如 "1,2,3"
	Data         string `json:"data"`         // 待签名数据
	Filename     string `json:"filename"`     // 签名文件名
	EncryptedKey string `json:"encryptedKey"` // base64编码的加密密钥
	UserName     string `json:"userName"`     // 用户名
	Address      string `json:"address"`      // 账户地址
	Signature    string `json:"signature"`    // base64编码的安全芯片签名
}

// 签名响应结构体
type SignResponse struct {
	Success   bool   `json:"success"`
	Signature string `json:"signature"`
}

// 创建签名会话密钥
func CreateSignSessionKey(token string, initiator string) (string, error) {
	// 发送请求创建密钥生成会话
	url := fmt.Sprintf("%s/sign/create/%s", BaseURL, initiator)

	// 创建请求
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", token)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
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
		return "", fmt.Errorf("创建签名会话失败，状态码: %d, 响应: %s", resp.StatusCode, body)
	}

	// 解析响应
	var response CreateSignSessionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	return response.SessionKey, nil
}

// GetSignAvailableUsers 获取可用的签名参与者列表
func GetSignAvailableUsers(token string, address string) ([]string, error) {
	// 创建请求
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/sign/users/%s", BaseURL, address),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 设置请求头
	req.Header.Set("Authorization", token)

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取签名用户列表失败，状态码: %d, 响应: %s", resp.StatusCode, body)
	}

	// 解析响应
	var response struct {
		Code int `json:"code"`
		Data []struct {
			Username string `json:"username"`
			Role     string `json:"role"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 提取参与者用户名
	participants := []string{}
	for _, user := range response.Data {
		participants = append(participants, user.Username)
	}

	return participants, nil
}

// 调用MPC签名API
func CallMpcSign(parties string, data string, filename string, username string, address string, encryptedKey string, sign string) (string, error) {
	// 构建请求体
	// 注意：在实际应用中，应当从安全芯片或其他安全存储中获取这些值
	signRequest := SignRequest{
		Parties:      parties,
		Data:         data,
		Filename:     filename,
		EncryptedKey: encryptedKey,
		UserName:     username,
		Address:      address,
		Signature:    sign,
	}

	requestBody, err := json.Marshal(signRequest)
	if err != nil {
		return "", fmt.Errorf("构建请求体失败: %v", err)
	}

	// 发送签名请求
	resp, err := http.Post(
		fmt.Sprintf("%s/api/v1/mpc/sign", MpcBaseURL),
		"application/json",
		bytes.NewBuffer(requestBody),
	)
	if err != nil {
		return "", fmt.Errorf("发送签名请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("签名失败，状态码: %d, 响应: %s", resp.StatusCode, body)
	}

	// 解析响应
	var response SignResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	if !response.Success {
		return "", fmt.Errorf("签名操作未成功")
	}

	return response.Signature, nil
}

// 处理协调者签名流程
func HandleCoordinatorSignFlow(coordinator *WSClient, sessionKey string, participants []string, address string, t *testing.T) {
	// 发送签名请求消息
	signRequestMsg := SignRequestMessage{
		BaseMessage:  BaseMessage{Type: MsgSignRequest},
		SessionKey:   sessionKey,
		Threshold:    2,
		TotalParts:   3,
		Data:         TestSignData,
		Address:      address,
		Participants: participants,
	}

	err := coordinator.SendMessage(signRequestMsg)
	if err != nil {
		t.Fatalf("发送签名请求失败: %v", err)
	}

	fmt.Printf("协调者 %s 发送签名请求: %s\n", coordinator.Username, sessionKey)

	// 等待签名完成消息
	fmt.Printf("协调者 %s 等待签名结果...\n", coordinator.Username)
	message, err := coordinator.ReadMessage()
	if err != nil {
		t.Fatalf("读取消息失败: %v", err)
	}

	// 解析基础消息类型
	var baseMsg BaseMessage
	if err := json.Unmarshal(message, &baseMsg); err != nil {
		t.Fatalf("解析基础消息失败: %v", err)
	}

	if baseMsg.Type == MsgSignComplete {
		var completeMsg SignCompleteMessage
		if err := json.Unmarshal(message, &completeMsg); err != nil {
			t.Fatalf("解析签名完成消息失败: %v", err)
		}

		fmt.Printf("签名完成! 会话: %s, 成功: %v, 签名结果: %s\n",
			completeMsg.SessionKey, completeMsg.Success, completeMsg.Signature)

		if !completeMsg.Success {
			t.Fatalf("签名失败: %s", completeMsg.Message)
		}
	} else if baseMsg.Type == MsgError {
		var errorMsg ErrorMessage
		if err := json.Unmarshal(message, &errorMsg); err != nil {
			t.Fatalf("解析错误消息失败: %v", err)
		}

		t.Fatalf("收到错误消息: %s, 详情: %s", errorMsg.Message, errorMsg.Details)
	} else {
		t.Fatalf("预期收到签名完成或错误消息，但收到: %s", baseMsg.Type)
	}
}

// 处理参与方签名流程
func HandleParticipantSignFlow(participant *WSClient, wg *sync.WaitGroup, t *testing.T) {
	defer wg.Done()

	// 侦听消息，处理签名邀请
	for {
		msgBytes, err := participant.ReadMessage()
		if err != nil {
			t.Fatalf("读取消息失败: %v", err)
		}

		var baseMsg BaseMessage
		if err := json.Unmarshal(msgBytes, &baseMsg); err != nil {
			t.Fatalf("解析基础消息失败: %v", err)
		}

		switch baseMsg.Type {
		case MsgSignInvite:
			// 处理签名邀请
			var inviteMsg SignInviteMessage
			if err := json.Unmarshal(msgBytes, &inviteMsg); err != nil {
				t.Fatalf("解析签名邀请消息失败: %v", err)
			}

			fmt.Printf("参与方 %s 收到签名邀请，会话: %s\n",
				participant.Username, inviteMsg.SessionKey)

			// 获取CPIC
			cpic, err := GetCPIC()
			if err != nil {
				t.Fatalf("获取CPIC失败: %v", err)
			}

			// 发送接受响应
			responseMsg := SignResponseMessage{
				BaseMessage: BaseMessage{Type: MsgSignResponse},
				SessionKey:  inviteMsg.SessionKey,
				PartIndex:   inviteMsg.PartIndex,
				CPIC:        cpic,
				Accept:      true,
			}

			err = participant.SendMessage(responseMsg)
			if err != nil {
				t.Fatalf("发送签名响应失败: %v", err)
			}

			fmt.Printf("参与方 %s 接受签名邀请，会话: %s\n",
				participant.Username, inviteMsg.SessionKey)

		case MsgSignParams:
			// 处理签名参数
			var paramsMsg SignParamsMessage
			if err := json.Unmarshal(msgBytes, &paramsMsg); err != nil {
				t.Fatalf("解析签名参数消息失败: %v", err)
			}

			fmt.Printf("参与方 %s 收到签名参数，会话: %s\n",
				participant.Username, paramsMsg.SessionKey)

			// 调用MPC签名接口
			signature, err := CallMpcSign(
				paramsMsg.Parties,
				paramsMsg.Data,
				paramsMsg.FileName,
				participant.Username,
				paramsMsg.Address,
				paramsMsg.EncryptedShard,
				paramsMsg.Signature,
			)
			if err != nil {
				// 发送失败结果
				resultMsg := SignResultMessage{
					BaseMessage: BaseMessage{Type: MsgSignResult},
					SessionKey:  paramsMsg.SessionKey,
					PartIndex:   paramsMsg.PartIndex,
					Success:     false,
					Message:     fmt.Sprintf("签名失败: %v", err),
				}

				_ = participant.SendMessage(resultMsg)
				t.Fatalf("MPC签名失败: %v", err)
			}

			// 发送成功结果
			resultMsg := SignResultMessage{
				BaseMessage: BaseMessage{Type: MsgSignResult},
				SessionKey:  paramsMsg.SessionKey,
				PartIndex:   paramsMsg.PartIndex,
				Success:     true,
				Signature:   signature,
				Message:     "签名成功",
			}

			err = participant.SendMessage(resultMsg)
			if err != nil {
				t.Fatalf("发送签名结果失败: %v", err)
			}

			fmt.Printf("参与方 %s 发送签名结果，会话: %s\n",
				participant.Username, paramsMsg.SessionKey)

			// 完成参与方流程
			return
		}
	}
}

// 测试签名流程
// 测试签名流程
func TestSignFlow(t *testing.T) {
	fmt.Println("\n===== 开始签名流程测试 =====")

	// 1. 登录所有预定义用户
	fmt.Println("1. 登录所有预定义用户...")
	err := LoginAllUsers()
	if err != nil {
		t.Fatalf("登录用户失败: %v", err)
	}

	// 2. 创建WebSocket客户端并连接
	fmt.Println("\n2. 创建WebSocket客户端并连接...")
	clients, err := CreateAllWSClients(TestUsers)
	if err != nil {
		t.Fatalf("创建WebSocket客户端失败: %v", err)
	}
	defer func() {
		for _, client := range clients {
			client.Close()
		}
	}()

	// 获取协调者客户端和参与者客户端
	coordinator := clients[0]
	participants := clients[1:]

	// 等待所有客户端完成注册
	fmt.Println("\n3. 等待所有客户端完成注册...")
	for _, client := range clients {
		if err := WaitForRegisterComplete(client, t); err != nil {
			t.Fatalf("等待客户端注册完成失败: %v", err)
		}
	}

	// 4. 创建签名会话
	fmt.Println("\n4. 创建签名会话...")
	sessionKey, err := CreateSignSessionKey(coordinator.Token, coordinator.Username)
	if err != nil {
		t.Fatalf("创建签名会话密钥失败: %v", err)
	}
	fmt.Printf("创建签名会话密钥成功: %s\n", sessionKey)

	// 5. 获取可用参与者列表
	fmt.Println("\n5. 获取可用签名参与者列表...")
	availableParticipants, err := GetSignAvailableUsers(coordinator.Token, address)
	if err != nil {
		t.Fatalf("获取可用签名参与者失败: %v", err)
	}
	fmt.Printf("可用签名参与者: %v\n", availableParticipants)

	// 确保我们有足够的参与者
	if len(availableParticipants) < 2 {
		t.Fatalf("没有足够的签名参与者可用，需要至少2个")
	}

	// 选择参与者
	selectedParticipants := availableParticipants
	if len(availableParticipants) > 3 {
		selectedParticipants = availableParticipants[:3]
	}

	// 6. 启动参与者处理协程
	fmt.Println("\n6. 启动参与者处理流程...")
	var wg sync.WaitGroup
	for _, participant := range participants {
		// 检查当前客户端是否是选定的参与者
		isSelected := false
		for _, p := range selectedParticipants {
			if participant.Username == p {
				isSelected = true
				break
			}
		}

		if isSelected {
			wg.Add(1)
			go HandleParticipantSignFlow(participant, &wg, t)
		}
	}

	// 7. 协调者发起签名请求并等待结果
	fmt.Println("\n7. 协调者发起签名请求...")
	wg.Add(1)
	go func() {
		defer wg.Done()
		HandleCoordinatorSignFlow(coordinator, sessionKey, selectedParticipants, address, t)
	}()

	// 等待所有参与者处理完成
	wg.Wait()

	fmt.Println("\n===== 签名流程测试完成 =====")
}

// 签名请求消息
type SignRequestMessage struct {
	BaseMessage
	SessionKey   string   `json:"session_key"`  // 会话唯一标识
	Threshold    int      `json:"threshold"`    // 门限值t
	TotalParts   int      `json:"total_parts"`  // 总分片数n
	Data         string   `json:"data"`         // 要签名的数据
	Address      string   `json:"address"`      // 账户地址
	Participants []string `json:"participants"` // 参与者用户名列表
}

// 定义签名相关消息结构
type SignCompleteMessage struct {
	BaseMessage
	SessionKey string `json:"session_key"` // 会话唯一标识
	Signature  string `json:"signature"`   // 最终签名结果
	Success    bool   `json:"success"`     // 签名是否成功
	Message    string `json:"message"`     // 成功或失败的消息
}

type SignInviteMessage struct {
	BaseMessage
	SessionKey   string   `json:"session_key"`  // 会话唯一标识
	Data         string   `json:"data"`         // 要签名的数据(32字节的哈希值)
	Address      string   `json:"address"`      // 账户地址
	PartIndex    int      `json:"part_index"`   // 参与者索引i
	SeID         string   `json:"se_id"`        // 安全芯片标识符
	Participants []string `json:"participants"` // 参与签名的所有用户名
}

type SignResponseMessage struct {
	BaseMessage
	SessionKey string `json:"session_key"`      // 会话唯一标识
	PartIndex  int    `json:"part_index"`       // 参与者索引i
	CPIC       string `json:"cpic"`             // 安全芯片唯一标识符
	Accept     bool   `json:"accept"`           // 是否接受参与
	Reason     string `json:"reason,omitempty"` // 拒绝原因(如果拒绝)
}

type SignParamsMessage struct {
	BaseMessage
	SessionKey     string `json:"session_key"`     // 会话唯一标识
	Data           string `json:"data"`            // 要签名的数据(Base64编码)
	Address        string `json:"address"`         // 账户地址
	Signature      string `json:"signature"`       // 用于从安全芯片中获取私钥分片的签名
	Parties        string `json:"parties"`         // 参与者列表(逗号分隔的索引)
	PartIndex      int    `json:"part_index"`      // 参与者索引i
	FileName       string `json:"filename"`        // 签名配置文件名
	EncryptedShard string `json:"encrypted_shard"` // Base64编码的加密密钥分片
}

type SignResultMessage struct {
	BaseMessage
	SessionKey string `json:"session_key"` // 会话唯一标识
	PartIndex  int    `json:"part_index"`  // 参与者索引i
	Success    bool   `json:"success"`     // 签名是否成功
	Signature  string `json:"signature"`   // 签名结果
	Message    string `json:"message"`     // 成功或失败的消息
}

// 错误消息结构体
type ErrorMessage struct {
	BaseMessage
	Message string `json:"message"`           // 错误消息
	Details string `json:"details,omitempty"` // 错误详情
}

// 定义本测试文件需要的常量
const (
	// 签名相关消息
	MsgSignRequest  MessageType = "sign_request"  // 协调方发送的签名请求
	MsgSignInvite   MessageType = "sign_invite"   // 服务器向参与方发送的签名邀请
	MsgSignResponse MessageType = "sign_response" // 参与方对签名邀请的响应
	MsgSignParams   MessageType = "sign_params"   // 服务器向参与方发送的签名参数
	MsgSignResult   MessageType = "sign_result"   // 参与方向服务器发送的签名结果
	MsgSignComplete MessageType = "sign_complete" // 服务器向协调方发送的签名完成消息

	// 错误消息
	MsgError MessageType = "error" // 错误消息
)
