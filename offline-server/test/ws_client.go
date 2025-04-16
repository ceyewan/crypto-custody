package test

import (
	"fmt"
	"log"
	"net/url"
	"offline-server/ws"
	"time"

	"github.com/gorilla/websocket"
)

// 从user_roles_test.go文件使用的类型定义
// 如果没有显式添加，由于同一个包，编译器会自动找到
// 但为了代码清晰，我们在这里明确声明

// LoginRequest 和 LoginResponse 结构定义在同包中的user_roles_test.go文件
// 可以直接使用

// WSClient WebSocket客户端结构
type WSClient struct {
	Conn     *websocket.Conn
	UserID   string
	Role     string
	Token    string
	Messages chan ws.Message
}

// 创建新的WebSocket客户端
func NewWSClient(userID, role string) (*WSClient, error) {
	u := url.URL{Scheme: "ws", Host: "localhost:8081", Path: "/ws"}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}

	client := &WSClient{
		Conn:     conn,
		UserID:   userID,
		Role:     role,
		Messages: make(chan ws.Message, 100),
	}

	// 启动消息接收协程
	go client.receiveMessages()

	return client, nil
}

// 接收消息
func (c *WSClient) receiveMessages() {
	defer close(c.Messages)

	for {
		var msg ws.Message
		err := c.Conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("客户端 %s 读取消息错误: %v", c.UserID, err)
			return
		}

		log.Printf("客户端 %s 收到消息: %+v", c.UserID, msg)
		c.Messages <- msg
	}
}

// 发送消息
func (c *WSClient) SendMessage(msg ws.Message) error {
	msg.UserID = c.UserID
	if c.Token != "" {
		msg.Token = c.Token
	}

	return c.Conn.WriteJSON(msg)
}

// 注册
func (c *WSClient) Register() error {
	msg := ws.Message{
		Type:   ws.RegisterMsg,
		UserID: c.UserID,
		Token:  c.Token,
		Payload: ws.RegisterPayload{
			UserID: c.UserID,
			Role:   c.Role,
		},
	}
	// 打印 msg 用于调试
	log.Printf("客户端 %s 发送注册消息: %+v", c.UserID, msg)
	return c.SendMessage(msg)
}

// 设置Token
func (c *WSClient) SetToken(token string) {
	c.Token = token
}

// 登录并设置Token
func (c *WSClient) LoginAndSetToken(username, password string) error {
	loginReq := LoginRequest{
		Username: username,
		Password: password,
	}

	loginResp, err := LoginUser(loginReq)
	if err != nil {
		return fmt.Errorf("登录失败: %w", err)
	}

	c.Token = loginResp.Token
	c.UserID = loginResp.User.Username
	return nil
}

// 等待特定类型的消息
func (c *WSClient) WaitForMessage(msgType ws.MessageType, timeout time.Duration) (ws.Message, bool) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for {
		select {
		case msg := <-c.Messages:
			if msg.Type == msgType {
				return msg, true
			}
		case <-timer.C:
			return ws.Message{}, false
		}
	}
}
