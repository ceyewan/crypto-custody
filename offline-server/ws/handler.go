package ws

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gorilla/websocket"
)

// MessageHandler 处理所有WebSocket消息
// 提供了消息分发和处理的功能，负责维护与客户端的通信
type MessageHandler struct {
	store Storage // 存储状态和客户端连接的接口
}

// NewMessageHandler 创建并初始化一个新的消息处理器
// 参数:
//   - store: 用于存储状态和客户端连接的存储接口
//
// 返回:
//   - 初始化后的MessageHandler指针
func NewMessageHandler(store Storage) *MessageHandler {
	return &MessageHandler{
		store: store,
	}
}

// HandleMessage 根据消息类型分发并处理接收到的WebSocket消息
// 参数:
//   - conn: 发送消息的WebSocket连接
//   - msg: 收到的消息对象
//
// 返回:
//   - error: 处理过程中遇到的错误
func (h *MessageHandler) HandleMessage(conn *websocket.Conn, msg Message) error {
	switch msg.Type {
	case RegisterMsg:
		return h.handleRegister(conn, msg)
	case KeyGenRequestMsg:
		return HandleKeyGenRequest(h.store, msg)
	case KeyGenResponseMsg:
		return HandleKeyGenResponse(h.store, msg)
	case KeyGenCompleteMsg:
		return HandleKeyGenComplete(h.store, msg)
	case SignRequestMsg:
		return HandleSignRequest(h.store, msg)
	case SignResponseMsg:
		return HandleSignResponse(h.store, msg)
	case SignResultMsg:
		return HandleSignResult(h.store, msg)
	default:
		err := fmt.Errorf("未知消息类型: %s", msg.Type)
		log.Printf("[ERROR] %v", err)
		return err
	}
}

// handleRegister 处理客户端注册请求
// 解析注册信息，将客户端添加到存储中，并发送确认消息
// 参数:
//   - conn: 发送注册请求的WebSocket连接
//   - msg: 包含注册信息的消息对象
//
// 返回:
//   - error: 处理过程中遇到的错误
func (h *MessageHandler) handleRegister(conn *websocket.Conn, msg Message) error {
	var payload RegisterPayload
	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		log.Printf("[ERROR] 序列化注册载荷失败: %v", err)
		return fmt.Errorf("序列化注册载荷失败: %w", err)
	}

	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		log.Printf("[ERROR] 解析注册载荷失败: %v", err)
		return fmt.Errorf("解析注册载荷失败: %w", err)
	}

	userID := payload.UserID
	role := payload.Role

	// 验证必要字段
	if userID == "" {
		err := fmt.Errorf("注册失败: 缺少用户ID")
		log.Printf("[ERROR] %v", err)
		if err := sendErrorMessage(conn, "缺少用户ID"); err != nil {
			log.Printf("[ERROR] 发送错误消息失败: %v", err)
		}
		return err
	}

	h.store.AddClient(userID, conn)
	h.store.SetClientRole(userID, role)

	log.Printf("[INFO] 客户端 %s 已注册为 %s", userID, role)

	// 发送确认消息
	response := Message{
		Type:    RegisterConfirmMsg,
		UserID:  userID,
		Payload: map[string]string{"status": "success"},
	}
	if err := SendMessage(conn, response); err != nil {
		log.Printf("[ERROR] 发送注册确认消息失败: %v", err)
		return fmt.Errorf("发送注册确认消息失败: %w", err)
	}
	return nil
}

// sendErrorMessage 向客户端发送错误消息
// 参数:
//   - conn: 要发送消息的WebSocket连接
//   - errorMsg: 错误信息
//
// 返回:
//   - error: 发送过程中遇到的错误
func sendErrorMessage(conn *websocket.Conn, errorMsg string) error {
	errResponse := Message{
		Type:    ErrorMsg,
		Payload: map[string]string{"error": errorMsg},
	}
	if err := SendMessage(conn, errResponse); err != nil {
		log.Printf("[ERROR] 发送错误消息失败: %v", err)
		return fmt.Errorf("发送错误消息失败: %w", err)
	}
	return nil
}
