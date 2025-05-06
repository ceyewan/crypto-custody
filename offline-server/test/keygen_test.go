package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocket消息类型
type MessageType string

const (
	// 消息类型常量
	MsgKeyGenRequest  MessageType = "keygen_request"  // 协调方发送的密钥生成请求
	MsgKeyGenInvite   MessageType = "keygen_invite"   // 服务器向参与方发送的密钥生成邀请
	MsgKeyGenResponse MessageType = "keygen_response" // 参与方对密钥生成邀请的响应
	MsgKeyGenParams   MessageType = "keygen_params"   // 服务器向参与方发送的密钥生成参数
	MsgKeyGenResult   MessageType = "keygen_result"   // 参与方向服务器发送的密钥生成结果
	MsgKeyGenComplete MessageType = "keygen_complete" // 服务器向协调方确认密钥生成完成
	MsgRegister       MessageType = "register"        // 客户端向服务器注册身份
)

// 基础消息结构
type BaseMessage struct {
	Type MessageType `json:"type"` // 消息类型
}

// 注册消息
type RegisterMessage struct {
	BaseMessage
	Username string `json:"username"` // 用户名
	Role     string `json:"role"`     // 用户角色
	Token    string `json:"token"`    // JWT令牌
}

// 密钥生成请求消息
type KeyGenRequestMessage struct {
	BaseMessage
	SessionKey   string   `json:"session_key"`  // 会话唯一标识
	Threshold    int      `json:"threshold"`    // 门限值t
	TotalParts   int      `json:"total_parts"`  // 总分片数n
	Participants []string `json:"participants"` // 参与者用户名列表
}

// 密钥生成邀请消息
type KeyGenInviteMessage struct {
	BaseMessage
	SessionKey   string   `json:"session_key"`  // 会话唯一标识
	Coordinator  string   `json:"coordinator"`  // 发起协调者用户名
	Threshold    int      `json:"threshold"`    // 门限值t
	TotalParts   int      `json:"total_parts"`  // 总分片数n
	PartIndex    int      `json:"part_index"`   // 当前参与者索引i
	SeID         string   `json:"se_id"`        // 安全芯片标识符
	Participants []string `json:"participants"` // 所有参与者用户名列表
}

// 密钥生成响应消息
type KeyGenResponseMessage struct {
	BaseMessage
	SessionKey string `json:"session_key"`      // 会话唯一标识
	PartIndex  int    `json:"part_index"`       // 参与者索引i
	CPIC       string `json:"cpic"`             // 安全芯片唯一标识符
	Accept     bool   `json:"accept"`           // 是否接受参与
	Reason     string `json:"reason,omitempty"` // 拒绝原因(如果拒绝)
}

// WebSocket客户端
type WSClient struct {
	Conn     *websocket.Conn
	Username string
	Role     string
	Token    string
	StopChan chan struct{} // 用于停止心跳协程
}

// 可用用户列表响应
type AvailableUsersResponse struct {
	Code int `json:"code"`
	Data []struct {
		ID       int    `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
		Role     string `json:"role"`
	} `json:"data"`
}

// 会话密钥响应
type SessionKeyResponse struct {
	SessionKey string `json:"session_key"`
}

// 密钥生成参数消息
type KeyGenParamsMessage struct {
	BaseMessage
	SessionKey string `json:"session_key"` // 会话唯一标识
	Threshold  int    `json:"threshold"`   // 门限值
	TotalParts int    `json:"total_parts"` // 总分片数
	PartIndex  int    `json:"part_index"`  // 参与者索引i
	FileName   string `json:"filename"`    // 密钥生成配置文件名
}

// 密钥生成结果消息
type KeyGenResultMessage struct {
	BaseMessage
	SessionKey     string `json:"session_key"`     // 会话唯一标识
	PartIndex      int    `json:"part_index"`      // 参与者索引i
	Address        string `json:"address"`         // 生成的账户地址
	CPIC           string `json:"cpic"`            // 安全芯片唯一标识符
	EncryptedShard string `json:"encrypted_shard"` // Base64编码的加密密钥分片
	Success        bool   `json:"success"`         // 密钥生成是否成功
	Message        string `json:"message"`         // 成功或失败的消息
}

// 创建WebSocket客户端并连接
func CreateWSClient(user UserInfo) (*WSClient, error) {
	// 创建WebSocket URL
	u := url.URL{Scheme: "ws", Host: "localhost:8081", Path: "/ws"}
	fmt.Printf("连接到WebSocket服务器: %s\n", u.String())

	// 建立WebSocket连接
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}
	conn, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("连接WebSocket服务器失败: %v", err)
	}

	client := &WSClient{
		Conn:     conn,
		Username: user.Username,
		Role:     user.Role,
		Token:    user.Token,
		StopChan: make(chan struct{}),
	}

	// 发送注册消息
	registerMsg := RegisterMessage{
		BaseMessage: BaseMessage{Type: MsgRegister},
		Username:    user.Username,
		Role:        user.Role,
		Token:       user.Token,
	}

	err = client.SendMessage(registerMsg)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("发送注册消息失败: %v", err)
	}

	// 启动心跳协程
	go client.startHeartbeat()

	fmt.Printf("用户 %s 已连接到WebSocket服务器并注册\n", user.Username)
	return client, nil
}

// 启动心跳保活
func (c *WSClient) startHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 发送ping消息
			if err := c.Conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				fmt.Printf("发送心跳消息失败: %v\n", err)
				return
			}
		case <-c.StopChan:
			return
		}
	}
}

// 发送WebSocket消息
func (c *WSClient) SendMessage(msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %v", err)
	}

	if err := c.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return fmt.Errorf("发送消息失败: %v", err)
	}
	return nil
}

// 接收WebSocket消息
func (c *WSClient) ReadMessage() ([]byte, error) {
	_, message, err := c.Conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("读取消息失败: %v", err)
	}
	return message, nil
}

// 关闭WebSocket连接
func (c *WSClient) Close() {
	if c.Conn != nil {
		// 停止心跳协程
		close(c.StopChan)
		c.Conn.Close()
	}
}

// 创建密钥生成会话密钥
func CreateKeyGenSessionKey(token string, initiator string) (string, error) {
	// 创建请求
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/keygen/create/%s", BaseURL, initiator),
		nil,
	)
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
		return "", fmt.Errorf("创建会话密钥失败，状态码: %d, 响应: %s", resp.StatusCode, body)
	}

	// 解析响应
	var response SessionKeyResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	return response.SessionKey, nil
}

// 获取可用参与者列表
func GetAvailableUsers(token string) ([]string, error) {
	// 创建请求
	req, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/keygen/users", BaseURL),
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
		return nil, fmt.Errorf("获取用户列表失败，状态码: %d, 响应: %s", resp.StatusCode, body)
	}

	// 解析响应
	var response AvailableUsersResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 提取参与者用户名
	participants := []string{}
	for _, user := range response.Data {
		if user.Role == "participant" {
			participants = append(participants, user.Username)
		}
	}

	return participants, nil
}

// TestKeyGenFlow 测试完整的密钥生成流程
func TestKeyGenFlow(t *testing.T) {
	fmt.Println("===== 开始密钥生成测试 =====")

	// 定义用户信息（一个协调者和三个参与者）
	users := []UserInfo{
		{
			Username: "coordinator",
			Password: "password123",
			Email:    "coordinator@example.com",
			Role:     CoordinatorRole,
		},
		{
			Username: "participant1",
			Password: "password123",
			Email:    "participant1@example.com",
			Role:     ParticipantRole,
		},
		{
			Username: "participant2",
			Password: "password123",
			Email:    "participant2@example.com",
			Role:     ParticipantRole,
		},
		{
			Username: "participant3",
			Password: "password123",
			Email:    "participant3@example.com",
			Role:     ParticipantRole,
		},
	}

	// 1. 先登录管理员账号
	fmt.Println("1. 登录管理员账号...")
	adminLogin, err := LoginUser(AdminUsername, AdminPassword)
	if err != nil {
		t.Fatalf("管理员登录失败: %v", err)
	}
	adminToken := adminLogin.Token
	fmt.Printf("管理员登录成功, 令牌: %s\n", adminToken)

	// 2. 注册所有测试用户
	fmt.Println("\n2. 注册测试用户...")
	for i, user := range users {
		fmt.Printf("注册用户 %d/%d: %s...\n", i+1, len(users), user.Username)
		registerResp, err := RegisterUser(user)
		if err != nil {
			t.Errorf("注册用户 %s 失败: %v", user.Username, err)
			continue
		}
		fmt.Printf("用户 %s 注册成功, 默认角色: %s\n", user.Username, registerResp.User.Role)
	}

	// 3. 使用管理员权限更新用户角色
	fmt.Println("\n3. 更新用户角色...")
	time.Sleep(1 * time.Second) // 简单延迟，确保注册完成

	for i, user := range users {
		fmt.Printf("更新用户 %d/%d: %s 角色为 %s...\n", i+1, len(users), user.Username, user.Role)
		err := UpdateUserRole(adminToken, user.Username, user.Role)
		if err != nil {
			t.Errorf("更新用户 %s 角色失败: %v", user.Username, err)
			continue
		}
		fmt.Printf("用户 %s 角色已更新为 %s\n", user.Username, user.Role)
	}

	// 4. 测试用户登录并获取令牌
	fmt.Println("\n4. 测试用户登录并获取令牌...")
	for i, user := range users {
		fmt.Printf("登录用户 %d/%d: %s...\n", i+1, len(users), user.Username)
		loginResp, err := LoginUser(user.Username, user.Password)
		if err != nil {
			t.Errorf("登录用户 %s 失败: %v", user.Username, err)
			continue
		}
		fmt.Printf("用户 %s 登录成功, 当前角色: %s\n", user.Username, loginResp.User.Role)
		fmt.Printf("令牌: %s\n", loginResp.Token)

		// 更新用户的Token
		users[i].Token = loginResp.Token
	}

	// 5. 创建WebSocket客户端并连接
	fmt.Println("\n5. 创建WebSocket客户端并连接...")
	var clients []*WSClient
	for i, user := range users {
		fmt.Printf("为用户 %s 创建WebSocket客户端...\n", user.Username)
		client, err := CreateWSClient(user)
		if err != nil {
			t.Fatalf("为用户 %s 创建WebSocket客户端失败: %v", user.Username, err)
		}
		clients = append(clients, client)
		fmt.Printf("用户 %s WebSocket客户端创建成功，索引: %d\n", user.Username, i)
	}
	defer func() {
		for _, client := range clients {
			client.Close()
		}
	}()

	// 获取协调者客户端和参与者客户端
	coordinator := clients[0]
	participants := clients[1:]

	// 6. 创建密钥生成会话
	fmt.Println("\n6. 创建密钥生成会话...")
	sessionKey, err := CreateKeyGenSessionKey(coordinator.Token, coordinator.Username)
	if err != nil {
		t.Fatalf("创建会话密钥失败: %v", err)
	}
	fmt.Printf("创建会话密钥成功: %s\n", sessionKey)

	// 7. 获取可用参与者列表
	fmt.Println("\n7. 获取可用参与者列表...")
	participantUsernames, err := GetAvailableUsers(coordinator.Token)
	if err != nil {
		t.Fatalf("获取可用参与者列表失败: %v", err)
	}
	fmt.Printf("可用参与者: %v\n", participantUsernames)

	// 确保我们有足够的参与者
	if len(participantUsernames) < 3 {
		t.Fatalf("没有足够的参与者可用，需要至少3个")
	}

	// 选择前3个参与者
	selectedParticipants := participantUsernames
	if len(participantUsernames) > 3 {
		selectedParticipants = participantUsernames[:3]
	}

	// 8. 发起密钥生成请求
	fmt.Println("\n8. 发起密钥生成请求...")
	keyGenReq := KeyGenRequestMessage{
		BaseMessage:  BaseMessage{Type: MsgKeyGenRequest},
		SessionKey:   sessionKey,
		Threshold:    2, // 设置门限值为2
		TotalParts:   3, // 总分片数为3
		Participants: selectedParticipants,
	}

	if err := coordinator.SendMessage(keyGenReq); err != nil {
		t.Fatalf("发送密钥生成请求失败: %v", err)
	}
	fmt.Println("发送密钥生成请求成功")

	// 9. 等待参与者收到邀请并响应
	fmt.Println("\n9. 等待参与者收到邀请并处理...")

	// 为每个参与者创建一个接收消息的协程
	var wg sync.WaitGroup
	for i, participant := range participants {
		wg.Add(1)
		go func(index int, client *WSClient) {
			defer wg.Done()

			// 等待接收密钥生成邀请
			fmt.Printf("参与者 %s 等待接收邀请...\n", client.Username)

			// 首先处理注册成功响应
			regMessage, err := client.ReadMessage()
			if err != nil {
				t.Errorf("参与者 %s 读取注册响应消息失败: %v", client.Username, err)
				return
			}

			var regBaseMsg BaseMessage
			if err := json.Unmarshal(regMessage, &regBaseMsg); err != nil {
				t.Errorf("参与者 %s 解析注册响应消息失败: %v", client.Username, err)
				return
			}

			if regBaseMsg.Type != "register_complete" {
				t.Errorf("参与者 %s 预期收到注册完成消息，但收到: %v", client.Username, regBaseMsg.Type)
				return
			}

			fmt.Printf("参与者 %s 注册成功，等待密钥生成邀请...\n", client.Username)

			// 然后等待密钥生成邀请消息
			message, err := client.ReadMessage()
			if err != nil {
				t.Errorf("参与者 %s 读取邀请消息失败: %v", client.Username, err)
				return
			}

			// 解析邀请消息
			var inviteMsg KeyGenInviteMessage
			if err := json.Unmarshal(message, &inviteMsg); err != nil {
				t.Errorf("参与者 %s 解析邀请消息失败: %v", client.Username, err)
				return
			}

			if inviteMsg.Type != MsgKeyGenInvite {
				t.Errorf("参与者 %s 收到了非邀请消息: %v", client.Username, inviteMsg.Type)
				return
			}

			fmt.Printf("参与者 %s 收到密钥生成邀请，会话密钥: %s, 索引: %d\n",
				client.Username, inviteMsg.SessionKey, inviteMsg.PartIndex)

			// 获取CPIC值
			fmt.Printf("参与者 %s 正在获取CPIC值...\n", client.Username)
			cpic, err := GetCPIC()
			if err != nil {
				t.Errorf("参与者 %s 获取CPIC失败: %v", client.Username, err)
				return
			}
			fmt.Printf("参与者 %s 获取CPIC成功: %s\n", client.Username, cpic)

			// 发送接受响应
			responseMsg := KeyGenResponseMessage{
				BaseMessage: BaseMessage{Type: MsgKeyGenResponse},
				SessionKey:  inviteMsg.SessionKey,
				PartIndex:   inviteMsg.PartIndex,
				CPIC:        cpic,
				Accept:      true,
			}

			if err := client.SendMessage(responseMsg); err != nil {
				t.Errorf("参与者 %s 发送接受响应失败: %v", client.Username, err)
				return
			}
			fmt.Printf("参与者 %s 已接受密钥生成邀请\n", client.Username)

			// 继续监听消息处理密钥生成参数等
			fmt.Printf("参与者 %s 等待后续消息...\n", client.Username)

			// 接收密钥生成参数消息
			paramsMessage, err := client.ReadMessage()
			if err != nil {
				t.Errorf("参与者 %s 读取参数消息失败: %v", client.Username, err)
				return
			}

			// 解析参数消息
			var paramsMsg KeyGenParamsMessage
			if err := json.Unmarshal(paramsMessage, &paramsMsg); err != nil {
				t.Errorf("参与者 %s 解析参数消息失败: %v", client.Username, err)
				return
			}

			if paramsMsg.Type != MsgKeyGenParams {
				t.Errorf("参与者 %s 预期收到参数消息，但收到: %v", client.Username, paramsMsg.Type)
				return
			}

			fmt.Printf("参与者 %s 收到密钥生成参数，会话密钥: %s, 索引: %d, 文件名: %s\n",
				client.Username, paramsMsg.SessionKey, paramsMsg.PartIndex, paramsMsg.FileName)

			// 调用MPC密钥生成API
			fmt.Printf("参与者 %s 正在调用密钥生成API...\n", client.Username)
			address, encryptedKey, err := CallMpcKeyGen(
				paramsMsg.Threshold,
				paramsMsg.TotalParts,
				paramsMsg.PartIndex,
				paramsMsg.FileName,
				client.Username,
			)
			if err != nil {
				t.Errorf("参与者 %s 调用密钥生成API失败: %v", client.Username, err)
				return
			}
			fmt.Printf("参与者 %s 密钥生成成功，地址: %s\n", client.Username, address)

			// 获取CPIC值用于返回结果
			var newCpic string
			newCpic, err = GetCPIC()
			if err != nil {
				t.Errorf("参与者 %s 获取CPIC失败: %v", client.Username, err)
				return
			}
			fmt.Printf("参与者 %s 获取CPIC成功: %s\n", client.Username, newCpic)

			// 发送密钥生成结果
			resultMsg := KeyGenResultMessage{
				BaseMessage:    BaseMessage{Type: MsgKeyGenResult},
				SessionKey:     paramsMsg.SessionKey,
				PartIndex:      paramsMsg.PartIndex,
				Address:        address,
				CPIC:           newCpic,
				EncryptedShard: encryptedKey,
				Success:        true,
				Message:        "密钥生成成功",
			}

			if err := client.SendMessage(resultMsg); err != nil {
				t.Errorf("参与者 %s 发送结果消息失败: %v", client.Username, err)
				return
			}
			fmt.Printf("参与者 %s 已发送密钥生成结果\n", client.Username)
		}(i, participant)
	}

	// 等待协调者的完成消息
	go func() {
		// 首先处理注册成功响应
		regMessage, err := coordinator.ReadMessage()
		if err != nil {
			t.Errorf("协调者 %s 读取注册响应消息失败: %v", coordinator.Username, err)
			return
		}

		var regBaseMsg BaseMessage
		if err := json.Unmarshal(regMessage, &regBaseMsg); err != nil {
			t.Errorf("协调者 %s 解析注册响应消息失败: %v", coordinator.Username, err)
			return
		}

		if regBaseMsg.Type != "register_complete" {
			t.Errorf("协调者 %s 预期收到注册完成消息，但收到: %v", coordinator.Username, regBaseMsg.Type)
			return
		}

		fmt.Printf("协调者 %s 注册成功，等待密钥生成结果...\n", coordinator.Username)

		// 等待接收密钥生成完成消息
		fmt.Printf("协调者 %s 等待密钥生成结果...\n", coordinator.Username)
		message, err := coordinator.ReadMessage()
		if err != nil {
			t.Errorf("协调者 %s 读取完成消息失败: %v", coordinator.Username, err)
			return
		}

		// 解析完成消息
		var baseMsg BaseMessage
		if err := json.Unmarshal(message, &baseMsg); err != nil {
			t.Errorf("协调者 %s 解析消息失败: %v", coordinator.Username, err)
			return
		}

		fmt.Printf("协调者 %s 收到消息类型: %s\n", coordinator.Username, baseMsg.Type)

		// 这里可以进一步解析完成消息的具体内容
		if baseMsg.Type == MsgKeyGenComplete {
			var completeMsg struct {
				BaseMessage
				SessionKey string `json:"session_key"`
				Address    string `json:"address"`
				Success    bool   `json:"success"`
				Message    string `json:"message"`
			}
			if err := json.Unmarshal(message, &completeMsg); err != nil {
				t.Errorf("协调者 %s 解析完成消息失败: %v", coordinator.Username, err)
				return
			}
			fmt.Printf("密钥生成完成! 地址: %s, 成功: %v, 消息: %s\n",
				completeMsg.Address, completeMsg.Success, completeMsg.Message)
		}
	}()

	// 等待所有参与者处理完成
	wg.Wait()

	fmt.Println("\n===== 密钥生成测试完成 =====")
}

// 调用MPC密钥生成API
func CallMpcKeyGen(threshold, parties, index int, filename, username string) (string, string, error) {
	// 构建请求体
	requestBody, err := json.Marshal(map[string]interface{}{
		"threshold": threshold,
		"parties":   parties,
		"index":     index,
		"filename":  filename,
		"username":  username,
	})
	if err != nil {
		return "", "", fmt.Errorf("构建请求体失败: %v", err)
	}

	// 发送POST请求
	resp, err := http.Post(
		"http://localhost:8088/api/v1/mpc/keygen",
		"application/json",
		bytes.NewBuffer(requestBody),
	)
	if err != nil {
		return "", "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("密钥生成API调用失败，状态码: %d, 响应: %s", resp.StatusCode, body)
	}

	// 解析响应
	var response struct {
		Success      bool   `json:"success"`
		UserName     string `json:"userName"`
		Address      string `json:"address"`
		EncryptedKey string `json:"encryptedKey"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return "", "", fmt.Errorf("解析响应失败: %v", err)
	}

	if !response.Success {
		return "", "", fmt.Errorf("密钥生成API返回失败状态")
	}

	return response.Address, response.EncryptedKey, nil
}
