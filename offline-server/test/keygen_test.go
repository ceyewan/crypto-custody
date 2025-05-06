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

	// 密钥生成参数常量
	KeyGenThreshold  = 2 // 门限值t
	KeyGenTotalParts = 3 // 总分片数n

	// 注册完成消息类型
	MsgRegisterComplete MessageType = "register_complete"
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

// 密钥生成完成消息
type KeyGenCompleteMessage struct {
	BaseMessage
	SessionKey string `json:"session_key"` // 会话唯一标识
	Address    string `json:"address"`     // 生成的账户地址
	Success    bool   `json:"success"`     // 密钥生成是否成功
	Message    string `json:"message"`     // 成功或失败的消息
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

// 为所有用户创建WebSocket客户端
func CreateAllWSClients(users []UserInfo) ([]*WSClient, error) {
	var clients []*WSClient

	for i, user := range users {
		fmt.Printf("为用户 %s 创建WebSocket客户端...\n", user.Username)
		client, err := CreateWSClient(user)
		if err != nil {
			// 关闭已创建的客户端
			for _, c := range clients {
				c.Close()
			}
			return nil, fmt.Errorf("为用户 %s 创建WebSocket客户端失败: %v", user.Username, err)
		}
		clients = append(clients, client)
		fmt.Printf("用户 %s WebSocket客户端创建成功，索引: %d\n", user.Username, i)
	}

	return clients, nil
}

// 等待并验证注册完成消息
func WaitForRegisterComplete(client *WSClient, t *testing.T) error {
	message, err := client.ReadMessage()
	if err != nil {
		return fmt.Errorf("读取注册响应消息失败: %v", err)
	}

	var regBaseMsg BaseMessage
	if err := json.Unmarshal(message, &regBaseMsg); err != nil {
		return fmt.Errorf("解析注册响应消息失败: %v", err)
	}

	if regBaseMsg.Type != MsgRegisterComplete {
		return fmt.Errorf("预期收到注册完成消息，但收到: %v", regBaseMsg.Type)
	}

	fmt.Printf("用户 %s 注册成功\n", client.Username)
	return nil
}

// 处理参与者的密钥生成流程
func HandleParticipantKeyGenFlow(participant *WSClient, wg *sync.WaitGroup, t *testing.T) {
	defer wg.Done()

	// 等待注册完成
	if err := WaitForRegisterComplete(participant, t); err != nil {
		t.Errorf("参与者 %s: %v", participant.Username, err)
		return
	}

	fmt.Printf("参与者 %s 等待密钥生成邀请...\n", participant.Username)

	// 等待密钥生成邀请消息
	message, err := participant.ReadMessage()
	if err != nil {
		t.Errorf("参与者 %s 读取邀请消息失败: %v", participant.Username, err)
		return
	}

	// 解析邀请消息
	var inviteMsg KeyGenInviteMessage
	if err := json.Unmarshal(message, &inviteMsg); err != nil {
		t.Errorf("参与者 %s 解析邀请消息失败: %v", participant.Username, err)
		return
	}

	if inviteMsg.Type != MsgKeyGenInvite {
		t.Errorf("参与者 %s 收到了非邀请消息: %v", participant.Username, inviteMsg.Type)
		return
	}

	fmt.Printf("参与者 %s 收到密钥生成邀请，会话密钥: %s, 索引: %d\n",
		participant.Username, inviteMsg.SessionKey, inviteMsg.PartIndex)

	// 获取CPIC值
	fmt.Printf("参与者 %s 正在获取CPIC值...\n", participant.Username)
	cpic, err := GetCPIC()
	if err != nil {
		t.Errorf("参与者 %s 获取CPIC失败: %v", participant.Username, err)
		return
	}

	// 发送接受响应
	responseMsg := KeyGenResponseMessage{
		BaseMessage: BaseMessage{Type: MsgKeyGenResponse},
		SessionKey:  inviteMsg.SessionKey,
		PartIndex:   inviteMsg.PartIndex,
		CPIC:        cpic,
		Accept:      true,
	}

	if err := participant.SendMessage(responseMsg); err != nil {
		t.Errorf("参与者 %s 发送接受响应失败: %v", participant.Username, err)
		return
	}
	fmt.Printf("参与者 %s 已接受密钥生成邀请\n", participant.Username)

	// 接收密钥生成参数消息
	paramsMessage, err := participant.ReadMessage()
	if err != nil {
		t.Errorf("参与者 %s 读取参数消息失败: %v", participant.Username, err)
		return
	}

	// 解析参数消息
	var paramsMsg KeyGenParamsMessage
	if err := json.Unmarshal(paramsMessage, &paramsMsg); err != nil {
		t.Errorf("参与者 %s 解析参数消息失败: %v", participant.Username, err)
		return
	}

	if paramsMsg.Type != MsgKeyGenParams {
		t.Errorf("参与者 %s 预期收到参数消息，但收到: %v", participant.Username, paramsMsg.Type)
		return
	}

	fmt.Printf("参与者 %s 收到密钥生成参数，会话密钥: %s, 索引: %d, 文件名: %s\n",
		participant.Username, paramsMsg.SessionKey, paramsMsg.PartIndex, paramsMsg.FileName)

	// 调用MPC密钥生成API
	fmt.Printf("参与者 %s 正在调用密钥生成API...\n", participant.Username)
	address, encryptedKey, err := CallMpcKeyGen(
		paramsMsg.Threshold,
		paramsMsg.TotalParts,
		paramsMsg.PartIndex,
		paramsMsg.FileName,
		participant.Username,
	)
	if err != nil {
		t.Errorf("参与者 %s 调用密钥生成API失败: %v", participant.Username, err)
		return
	}
	fmt.Printf("参与者 %s 密钥生成成功，地址: %s\n", participant.Username, address)

	// 获取新的CPIC值用于返回结果
	newCpic, err := GetCPIC()
	if err != nil {
		t.Errorf("参与者 %s 获取CPIC失败: %v", participant.Username, err)
		return
	}

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

	if err := participant.SendMessage(resultMsg); err != nil {
		t.Errorf("参与者 %s 发送结果消息失败: %v", participant.Username, err)
		return
	}
	fmt.Printf("参与者 %s 已发送密钥生成结果\n", participant.Username)
}

// 处理协调者的密钥生成流程
func HandleCoordinatorKeyGenFlow(coordinator *WSClient, sessionKey string, participants []string, t *testing.T) {
	// 等待注册完成
	if err := WaitForRegisterComplete(coordinator, t); err != nil {
		t.Errorf("协调者 %s: %v", coordinator.Username, err)
		return
	}

	// 发起密钥生成请求
	keyGenReq := KeyGenRequestMessage{
		BaseMessage:  BaseMessage{Type: MsgKeyGenRequest},
		SessionKey:   sessionKey,
		Threshold:    KeyGenThreshold,
		TotalParts:   KeyGenTotalParts,
		Participants: participants,
	}

	if err := coordinator.SendMessage(keyGenReq); err != nil {
		t.Errorf("协调者 %s 发送密钥生成请求失败: %v", coordinator.Username, err)
		return
	}
	fmt.Printf("协调者 %s 已发送密钥生成请求\n", coordinator.Username)

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

	if baseMsg.Type != MsgKeyGenComplete {
		t.Errorf("协调者 %s 预期收到密钥生成完成消息，但收到: %s", coordinator.Username, baseMsg.Type)
		return
	}

	// 解析具体结果
	var completeMsg KeyGenCompleteMessage
	if err := json.Unmarshal(message, &completeMsg); err != nil {
		t.Errorf("协调者 %s 解析完成消息失败: %v", coordinator.Username, err)
		return
	}

	fmt.Printf("密钥生成完成! 地址: %s, 成功: %v, 消息: %s\n",
		completeMsg.Address, completeMsg.Success, completeMsg.Message)
}

// TestKeyGenFlow 测试完整的密钥生成流程
func TestKeyGenFlow(t *testing.T) {
	fmt.Println("===== 开始密钥生成测试 =====")

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

	// 3. 创建密钥生成会话
	fmt.Println("\n3. 创建密钥生成会话...")
	sessionKey, err := CreateKeyGenSessionKey(coordinator.Token, coordinator.Username)
	if err != nil {
		t.Fatalf("创建会话密钥失败: %v", err)
	}
	fmt.Printf("创建会话密钥成功: %s\n", sessionKey)

	// 4. 获取可用参与者列表
	fmt.Println("\n4. 获取可用参与者列表...")
	participantUsernames, err := GetAvailableUsers(coordinator.Token)
	if err != nil {
		t.Fatalf("获取可用参与者列表失败: %v", err)
	}
	fmt.Printf("可用参与者: %v\n", participantUsernames)

	// 确保我们有足够的参与者
	if len(participantUsernames) < KeyGenTotalParts {
		t.Fatalf("没有足够的参与者可用，需要至少%d个", KeyGenTotalParts)
	}

	// 选择前N个参与者
	selectedParticipants := participantUsernames
	if len(participantUsernames) > KeyGenTotalParts {
		selectedParticipants = participantUsernames[:KeyGenTotalParts]
	}

	// 5. 启动参与者处理协程
	fmt.Println("\n5. 启动参与者处理流程...")
	var wg sync.WaitGroup
	for _, participant := range participants {
		wg.Add(1)
		go HandleParticipantKeyGenFlow(participant, &wg, t)
	}

	// 6. 协调者发起密钥生成请求并等待结果
	fmt.Println("\n6. 协调者发起密钥生成请求...")
	wg.Add(1)
	go func() {
		defer wg.Done()
		HandleCoordinatorKeyGenFlow(coordinator, sessionKey, selectedParticipants, t)
	}()

	// 等待所有参与者处理完成
	wg.Wait()

	fmt.Println("\n===== 密钥生成测试完成 =====")
}
