package test

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"offline-server/ws"
)

// 生成模拟的密钥分享JSON
func generateMockShareJSON(keyID string, partIndex int) string {
	// 这里生成一个简单的模拟JSON字符串
	return fmt.Sprintf(`{"key_id":"%s","part_index":%d,"data":"mock_share_data"}`, keyID, partIndex)
}

// 测试密钥生成流程
func TestKeyGenProcess(t *testing.T) {
	// 创建一个唯一的密钥标识
	keyID := fmt.Sprintf("test-key-%d", time.Now().Unix())
	t.Logf("使用密钥标识: %s", keyID)

	// 用户IDs和角色
	coordinator := "coordinator_user"
	participants := []string{"participant_user1", "participant_user2", "participant_user3"}

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
	participantClients := make([]*WSClient, len(participants))
	for i, userID := range participants {
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

		participantClients[i] = client
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

	for i, client := range participantClients {
		if err := client.Register(); err != nil {
			t.Fatalf("参与者 %s 注册失败: %v", client.UserID, err)
		}

		if _, ok := client.WaitForMessage(ws.RegisterConfirmMsg, 5*time.Second); !ok {
			t.Fatalf("等待参与者 %s 注册确认超时", client.UserID)
		} else {
			t.Logf("参与者 %d (%s) 注册成功", i+1, client.UserID)
		}
	}

	// 协调者发起密钥生成请求
	threshold := 1
	totalParts := 3

	t.Logf("协调者发起密钥生成请求: 阈值 = %d, 总分片数 = %d", threshold, totalParts)
	err = coordClient.SendMessage(ws.Message{
		Type: ws.KeyGenRequestMsg,
		// 移除UserID字段，因为它与Token中的UserID可能不匹配
		// 服务端会从Token中提取正确的UserID
		Token: coordClient.Token,
		Payload: ws.KeyGenRequestPayload{
			KeyID:        keyID,
			Threshold:    threshold,
			TotalParts:   totalParts,
			Participants: participants,
		},
	})
	if err != nil {
		t.Fatalf("发送密钥生成请求失败: %v", err)
	}

	// 参与者等待并接受邀请
	for i, client := range participantClients {
		t.Logf("等待参与者 %d (%s) 接收邀请", i+1, client.UserID)
		msg, ok := client.WaitForMessage(ws.KeyGenInviteMsg, 10*time.Second)
		if !ok {
			t.Fatalf("参与者 %d 等待邀请超时", i+1)
		}

		// 解析邀请载荷
		var payload map[string]interface{}
		payloadBytes, _ := json.Marshal(msg.Payload)
		json.Unmarshal(payloadBytes, &payload)

		partIndex := int(payload["part_index"].(float64))
		t.Logf("参与者 %d 收到邀请, 分片索引: %d", i+1, partIndex)

		// 发送接受响应
		err = client.SendMessage(ws.Message{
			Type: ws.KeyGenResponseMsg,
			// 不需要手动设置UserID，服务器会从Token中提取
			Token: client.Token,
			Payload: ws.KeyGenResponsePayload{
				KeyID:     keyID,
				PartIndex: partIndex,
				Response:  true, // 接受邀请
			},
		})
		if err != nil {
			t.Fatalf("参与者 %d 发送接受响应失败: %v", i+1, err)
		}
		t.Logf("参与者 %d 已接受邀请", i+1)
	}

	// 参与者等待密钥生成参数
	for i, client := range participantClients {
		t.Logf("等待参与者 %d 接收密钥生成参数", i+1)
		msg, ok := client.WaitForMessage(ws.KeyGenParamsMsg, 10*time.Second)
		if !ok {
			t.Fatalf("参与者 %d 等待密钥生成参数超时", i+1)
		}

		// 解析参数载荷
		var payload map[string]interface{}
		payloadBytes, _ := json.Marshal(msg.Payload)
		json.Unmarshal(payloadBytes, &payload)

		partIndex := int(payload["part_index"].(float64))
		outputFile := payload["output_file"].(string)
		t.Logf("参与者 %d 收到密钥生成参数, 分片索引: %d, 输出文件: %s", i+1, partIndex, outputFile)

		// 模拟密钥生成过程
		t.Logf("参与者 %d 执行密钥生成命令: ./gg20_keygen -t %d -n %d -i %d --output %s",
			i+1, threshold, totalParts, partIndex, outputFile)

		// 生成模拟的分享文件内容
		shareJSON := generateMockShareJSON(keyID, partIndex)

		// 生成账户地址（使用分享的哈希值）
		hasher := sha256.New()
		hasher.Write([]byte(shareJSON))
		accountAddr := hex.EncodeToString(hasher.Sum(nil))

		// 发送密钥生成完成消息
		err = client.SendMessage(ws.Message{
			Type:  ws.KeyGenCompleteMsg,
			Token: client.Token,
			Payload: ws.KeyGenCompletePayload{
				KeyID:       keyID,
				PartIndex:   partIndex,
				AccountAddr: accountAddr,
				ShareJSON:   shareJSON,
			},
		})
		if err != nil {
			t.Fatalf("参与者 %d 发送密钥生成完成消息失败: %v", i+1, err)
		}
		t.Logf("参与者 %d 已完成密钥生成", i+1)
	}

	// 协调者等待密钥生成确认
	t.Log("等待协调者收到密钥生成成功确认")
	msg, ok := coordClient.WaitForMessage(ws.KeyGenConfirmMsg, 15*time.Second)
	if !ok {
		t.Fatalf("协调者等待密钥生成确认超时")
	}

	// 解析确认载荷
	var payload map[string]interface{}
	payloadBytes, _ := json.Marshal(msg.Payload)
	json.Unmarshal(payloadBytes, &payload)

	status := payload["status"].(string)
	if status != "success" {
		t.Fatalf("密钥生成失败: %v", payload)
	}

	accountAddr := payload["account_addr"].(string)
	t.Logf("密钥生成成功! 账户地址: %s", accountAddr)
}
