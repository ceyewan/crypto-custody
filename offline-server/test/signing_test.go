package test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"offline-server/ws"

	"github.com/gorilla/websocket"
)

// 生成模拟的签名结果
func generateMockSignature(data string, keyID string, partIndex int) string {
	// 这里生成一个简单的模拟签名字符串
	return fmt.Sprintf("sig-%s-%s-%d", data, keyID, partIndex)
}

// 测试签名流程
func TestSigningProcess(t *testing.T) {
	// 使用与密钥生成测试相同的keyID，假设密钥已经生成
	keyID := "test-key-1744787198"
	accountAddr := "0x1234567890abcdef1234567890abcdef12345678"
	data := "hello" // 要签名的数据

	t.Logf("使用密钥标识: %s, 签名数据: %s", keyID, data)

	// 用户IDs和角色
	coordinator := "coordinator_user"
	allParticipants := []string{"participant_user1", "participant_user2", "participant_user3"}

	// 选择两个参与者参与签名 (模拟命令中的 -p 1,2)
	signingParticipants := []string{allParticipants[0], allParticipants[1]}

	// 首先通过Web服务登录获取Token
	t.Log("通过Web服务登录获取Token")

	// 创建协调者客户端
	coordClient, err := NewWSClient(coordinator, "coordinator")
	if err != nil {
		t.Fatalf("创建协调者客户端失败: %v", err)
	}
	defer coordClient.Conn.Close()

	// 协调者登录并获取Token
	err = coordClient.LoginAndSetToken(coordinator, "password123")
	if err != nil {
		t.Fatalf("协调者登录失败: %v", err)
	}
	t.Logf("协调者登录成功，获取Token")

	// 创建参与者客户端
	participantClients := make(map[string]*WSClient)
	for _, userID := range allParticipants {
		client, err := NewWSClient(userID, "participant")
		if err != nil {
			t.Fatalf("创建参与者客户端 %s 失败: %v", userID, err)
		}
		defer client.Conn.Close()

		// 参与者登录并获取Token
		err = client.LoginAndSetToken(userID, "password123")
		if err != nil {
			t.Fatalf("参与者 %s 登录失败: %v", userID, err)
		}
		t.Logf("参与者 %s 登录成功，获取Token", userID)

		participantClients[userID] = client
	}

	// 注册所有客户端
	if err := coordClient.Register(); err != nil {
		t.Fatalf("协调者注册失败: %v", err)
	}

	// 等待注册确认
	if _, ok := coordClient.WaitForMessage(ws.RegisterConfirmMsg, 5*time.Second); !ok {
		t.Fatalf("等待协调者注册确认超时")
	} else {
		t.Logf("协调者注册成功")
	}

	for userID, client := range participantClients {
		if err := client.Register(); err != nil {
			t.Fatalf("参与者 %s 注册失败: %v", userID, err)
		}

		if _, ok := client.WaitForMessage(ws.RegisterConfirmMsg, 5*time.Second); !ok {
			t.Fatalf("等待参与者 %s 注册确认超时", userID)
		} else {
			t.Logf("参与者 %s 注册成功", userID)
		}
	}

	// 协调者发起签名请求
	t.Logf("协调者发起签名请求: KeyID = %s, 数据 = %s", keyID, data)
	err = coordClient.SendMessage(ws.Message{
		Type: ws.SignRequestMsg,
		Payload: ws.SignRequestPayload{
			KeyID:        keyID,
			Data:         data,
			AccountAddr:  accountAddr,
			Participants: signingParticipants, // 指定参与签名的用户
		},
	})
	if err != nil {
		t.Fatalf("发送签名请求失败: %v", err)
	}

	// 参与者等待并接受签名邀请
	for _, userID := range signingParticipants {
		client := participantClients[userID]
		t.Logf("等待参与者 %s 接收签名邀请", userID)
		msg, ok := client.WaitForMessage(ws.SignInviteMsg, 10*time.Second)
		if !ok {
			t.Fatalf("参与者 %s 等待签名邀请超时", userID)
		}

		// 解析邀请载荷
		var payload map[string]interface{}
		payloadBytes, _ := json.Marshal(msg.Payload)
		json.Unmarshal(payloadBytes, &payload)

		partIndex := int(payload["part_index"].(float64))
		t.Logf("参与者 %s 收到签名邀请, 分片索引: %d", userID, partIndex)

		// 发送接受响应
		err = client.SendMessage(ws.Message{
			Type: ws.SignResponseMsg,
			Payload: ws.SignResponsePayload{
				KeyID:     keyID,
				PartIndex: partIndex,
				Response:  true, // 接受邀请
			},
		})
		if err != nil {
			t.Fatalf("参与者 %s 发送接受响应失败: %v", userID, err)
		}
		t.Logf("参与者 %s 已接受签名邀请", userID)
	}

	// 参与者等待签名参数
	for _, userID := range signingParticipants {
		client := participantClients[userID]
		t.Logf("等待参与者 %s 接收签名参数", userID)
		msg, ok := client.WaitForMessage(ws.SignParamsMsg, 10*time.Second)
		if !ok {
			t.Fatalf("参与者 %s 等待签名参数超时", userID)
		}

		// 解析参数载荷
		var payload map[string]interface{}
		payloadBytes, _ := json.Marshal(msg.Payload)
		json.Unmarshal(payloadBytes, &payload)

		partIndex := int(payload["part_index"].(float64))
		participants := payload["participants"].(string)
		// shareJSON := payload["share_json"].(string)

		t.Logf("参与者 %s 收到签名参数, 分片索引: %d, 参与者列表: %s",
			userID, partIndex, participants)

		// 模拟执行签名命令行工具
		outputFile := fmt.Sprintf("local-share%d.json", partIndex)
		t.Logf("参与者 %s 执行签名命令: ./gg20_signing -p %s -d \"%s\" -l %s",
			userID, participants, data, outputFile)

		// 生成模拟的签名结果
		signature := generateMockSignature(data, keyID, partIndex)

		// 发送签名结果
		err = client.SendMessage(ws.Message{
			Type: ws.SignResultMsg,
			Payload: ws.SignResultPayload{
				KeyID:     keyID,
				PartIndex: partIndex,
				Signature: signature,
			},
		})
		if err != nil {
			t.Fatalf("参与者 %s 发送签名结果失败: %v", userID, err)
		}
		t.Logf("参与者 %s 已发送签名结果", userID)
	}

	// 协调者等待签名完成确认
	t.Log("等待协调者收到签名完成确认")
	msg, ok := coordClient.WaitForMessage(ws.SignCompleteMsg, 15*time.Second)
	if !ok {
		t.Fatalf("协调者等待签名完成确认超时")
	}

	// 解析确认载荷
	var payload map[string]interface{}
	payloadBytes, _ := json.Marshal(msg.Payload)
	json.Unmarshal(payloadBytes, &payload)

	status := payload["status"].(string)
	if status != "success" {
		t.Fatalf("签名失败: %v", payload)
	}

	signature := payload["signature"].(string)
	t.Logf("签名成功! 签名结果: %s", signature)

	// 等待一段时间，确保所有消息都已处理完毕
	time.Sleep(1 * time.Second)

	// 优雅关闭连接
	closeConnections := func() {
		// 先关闭参与者连接
		for userID, client := range participantClients {
			t.Logf("关闭参与者 %s 的连接", userID)
			client.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			client.Conn.Close()
		}

		// 最后关闭协调者连接
		t.Log("关闭协调者的连接")
		coordClient.Conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		coordClient.Conn.Close()
	}

	// 执行优雅关闭
	closeConnections()
}
