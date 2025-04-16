package ws

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// Client 表示一个WebSocket客户端连接
// 负责处理与客户端的通信，包括接收消息、发送消息和断开连接处理
type Client struct {
	conn    *websocket.Conn // WebSocket连接
	handler *MessageHandler // 消息处理器
	store   Storage         // 存储接口
	userID  string          // 客户端用户ID
}

// NewClient 创建并初始化一个新的客户端实例
//
// 参数:
//   - conn: WebSocket连接对象
//   - handler: 用于处理消息的处理器
//   - store: 用于存储状态和客户端信息的存储接口
//
// 返回:
//   - 初始化后的Client对象
func NewClient(conn *websocket.Conn, handler *MessageHandler, store Storage) *Client {
	return &Client{
		conn:    conn,
		handler: handler,
		store:   store,
	}
}

// Listen 开始监听客户端消息
// 这是一个阻塞方法，应该在一个新的goroutine中运行
// 会持续监听客户端消息，直到连接关闭或出现错误
func (c *Client) Listen() {
	defer func() {
		c.conn.Close()
		c.handleDisconnect()
	}()

	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-heartbeatTicker.C:
			// 发送心跳包
			err := c.conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				log.Printf("发送心跳包失败: %v", err)
				return
			}
		default:
			// 读取消息
			_, msgBytes, err := c.conn.ReadMessage()
			if err != nil {
				// 处理连接关闭或错误
				log.Printf("客户端 %s 读取消息错误: %v", c.userID, err)
				return // 错误时直接返回，触发defer中的清理
			}

			// 解析消息
			var msg Message
			if err := json.Unmarshal(msgBytes, &msg); err != nil {
				log.Printf("解析消息失败: %v", err)
				continue
			}

			// 如果是注册消息，设置客户端ID
			if msg.Type == RegisterMsg {
				var payload RegisterPayload
				payloadBytes, err := json.Marshal(msg.Payload)
				if err != nil {
					log.Printf("序列化注册载荷失败: %v", err)
					continue
				}

				if err := json.Unmarshal(payloadBytes, &payload); err == nil {
					c.userID = payload.UserID
				}
			}

			// 处理消息
			c.handler.HandleMessage(c.conn, msg)
		}
	}
}

// handleDisconnect 处理客户端断开连接
// 从存储中移除客户端，并记录日志
func (c *Client) handleDisconnect() {
	if c.userID != "" {
		c.store.RemoveClient(c.userID)
		log.Printf("客户端 %s 已断开连接", c.userID)
	}
}

// SendMessage 向客户端发送WebSocket消息
//
// 参数:
//   - conn: 要发送消息的WebSocket连接
//   - msg: 要发送的消息对象
//
// 返回:
//   - error: 如果消息发送失败则返回错误，否则返回nil
func SendMessage(conn *websocket.Conn, msg Message) error {
	// 序列化消息
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("序列化消息失败: %v", err)
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	// 发送消息，最多重试3次
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		err := conn.WriteMessage(websocket.TextMessage, msgBytes)
		if err == nil {
			// 添加消息发送成功的日志
			log.Printf("成功发送消息 类型: %s, 用户: %s", msg.Type, msg.UserID)
			return nil
		}

		log.Printf("发送消息失败 (尝试 %d/%d): %v", i+1, maxRetries, err)
		if i < maxRetries-1 {
			// 重试前等待一小段时间
			time.Sleep(100 * time.Millisecond)
		}
	}

	log.Printf("发送消息最终失败")
	return fmt.Errorf("发送消息失败，已重试%d次", maxRetries)
}

// SendMessageToUser 向特定用户发送消息
//
// 参数:
//   - store: 存储接口，用于查找用户连接
//   - userID: 目标用户ID
//   - msg: 要发送的消息对象
//
// 返回:
//   - error: 如果消息发送失败则返回错误，否则返回nil
func SendMessageToUser(store Storage, userID string, msg Message) error {
	conn, exists := store.GetClient(userID)
	if !exists {
		log.Printf("找不到用户 %s 的连接", userID)
		return fmt.Errorf("找不到用户 %s 的连接", userID)
	}

	if wsConn, ok := conn.(*websocket.Conn); ok {
		return SendMessage(wsConn, msg)
	}

	log.Printf("用户 %s 的连接类型无效", userID)
	return fmt.Errorf("用户 %s 的连接类型无效", userID)
}
