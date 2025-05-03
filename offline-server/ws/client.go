package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"offline-server/tools"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// 缓冲区大小常量
const (
	ReadBufferSize  = 256 // 读缓冲区大小
	WriteBufferSize = 256 // 写缓冲区大小

	MaxMessageSize = 65536            // 最大消息大小(64KB)
	PongWait       = 60 * time.Second // pong等待时间
	PingPeriod     = 30 * time.Second // ping发送周期
	WriteWait      = 10 * time.Second // 写超时
)

// Client 表示单个WebSocket客户端连接
// 负责管理连接生命周期和消息处理
type Client struct {
	// 客户端标识
	username string     // 用户名
	role     ClientRole // 用户角色

	// 连接相关
	conn    *websocket.Conn // WebSocket连接
	hub     *Hub            // 客户端集线器
	handler *MessageHandler // 消息处理器

	// 消息通道
	readChan  chan []byte // 读取消息的缓冲通道
	writeChan chan []byte // 写入消息的缓冲通道

	// 协程控制
	readCtx     context.Context    // 读取协程上下文
	readCancel  context.CancelFunc // 读取协程取消函数
	writeCtx    context.Context    // 写入协程上下文
	writeCancel context.CancelFunc // 写入协程取消函数
	wg          sync.WaitGroup     // 等待协程组

	// 状态管理
	closed      bool         // 连接是否已关闭
	closedMutex sync.RWMutex // 保护closed字段的锁
	closeOnce   sync.Once    // 确保只关闭一次
}

// NewClient 创建并初始化新的客户端
func NewClient(conn *websocket.Conn, hub *Hub, handler *MessageHandler) *Client {
	// 创建上下文
	readCtx, readCancel := context.WithCancel(context.Background())
	writeCtx, writeCancel := context.WithCancel(context.Background())

	// 初始化客户端
	client := &Client{
		conn:        conn,
		hub:         hub,
		handler:     handler,
		readChan:    make(chan []byte, ReadBufferSize),
		writeChan:   make(chan []byte, WriteBufferSize),
		readCtx:     readCtx,
		readCancel:  readCancel,
		writeCtx:    writeCtx,
		writeCancel: writeCancel,
	}

	return client
}

// Start 启动客户端消息处理循环
func (c *Client) Start() {
	// 启动读写协程
	c.wg.Add(2)
	go c.readPump()
	go c.writePump()

	log.Printf("启动客户端处理")
}

// Username 获取客户端用户名
func (c *Client) Username() string {
	return c.username
}

// SetUsername 设置客户端用户名
func (c *Client) SetUsername(username string) {
	c.username = username
}

// Role 获取客户端角色
func (c *Client) Role() ClientRole {
	return c.role
}

// SetRole 设置客户端角色
func (c *Client) SetRole(role ClientRole) {
	c.role = role
}

// Hub 获取客户端所属的Hub
func (c *Client) Hub() *Hub {
	return c.hub
}

// readPump 处理读取WebSocket消息的协程
func (c *Client) readPump() {
	defer c.wg.Done()
	defer c.Close()

	// 设置连接参数
	c.conn.SetReadLimit(MaxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(PongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(PongWait))
		return nil
	})

	// 持续读取消息
	for {
		select {
		case <-c.readCtx.Done():
			// 上下文已取消，退出协程
			return
		default:
			// 读取消息
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err,
					websocket.CloseGoingAway,
					websocket.CloseAbnormalClosure) {
					log.Printf("读取错误: %v", err)
				}
				return
			}

			// 处理收到的消息
			if err := c.handleMessage(message); err != nil {
				log.Printf("处理客户端消息失败: %v", err)
				// 发送错误响应
				if errMsg := c.SendErrorMessage(err.Error(), ""); errMsg != nil {
					log.Printf("发送错误消息失败: %v", errMsg)
				}
			}
		}
	}
}

// writePump 处理发送WebSocket消息的协程
func (c *Client) writePump() {
	defer c.wg.Done()

	ticker := time.NewTicker(PingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-c.writeCtx.Done():
			// 上下文已取消，退出协程
			return
		case message, ok := <-c.writeChan:
			// 设置写入截止时间
			c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if !ok {
				// 通道已关闭
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 添加队列中的所有消息
			n := len(c.writeChan)
			for i := 0; i < n; i++ {
				w.Write(<-c.writeChan)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			// 发送ping消息
			c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage 处理收到的消息
func (c *Client) handleMessage(message []byte) error {
	// 首先解析基础消息以获取类型
	var baseMsg BaseMessage
	if err := json.Unmarshal(message, &baseMsg); err != nil {
		return fmt.Errorf("解析消息类型失败: %w", err)
	}

	// 根据消息类型进行不同处理
	switch baseMsg.Type {
	case MsgRegister:
		// 处理注册消息 - 需要验证Token
		var registerMsg RegisterMessage
		if err := json.Unmarshal(message, &registerMsg); err != nil {
			return fmt.Errorf("解析注册消息失败: %w", err)
		}
		return c.handleRegisterMessage(registerMsg)

	default:
		// 非注册消息 - 检查客户端是否已注册
		if c.username == "" {
			return fmt.Errorf("客户端尚未注册，请先发送注册消息")
		}

		// 交由消息处理器处理其他类型消息
		return c.handler.ProcessMessage(baseMsg.Type, message, c)
	}
}

// handleRegisterMessage 处理注册消息
func (c *Client) handleRegisterMessage(msg RegisterMessage) error {
	// 验证Token
	username, role, err := tools.ValidateToken(msg.Token)
	if err != nil {
		return fmt.Errorf("验证Token失败: %w", err)
	}

	// 验证Token中的信息与消息中的信息是否一致
	if username != msg.Username {
		return fmt.Errorf("token中的用户名与消息中的用户名不匹配")
	}
	if ClientRole(role) != msg.Role {
		return fmt.Errorf("token中的角色与消息中的角色不匹配")
	}

	// 设置客户端属性
	c.SetUsername(msg.Username)
	c.SetRole(msg.Role)

	// 注册到Hub
	c.hub.RegisterClient(msg.Username, c)

	// 发送确认消息
	confirmMsg := RegisterCompleteMessage{
		BaseMessage: BaseMessage{Type: MsgRegisterComplete},
		Success:     true,
		Message:     "注册成功",
	}

	return c.SendMessage(confirmMsg)
}

// SendMessage 发送消息
func (c *Client) SendMessage(msg Message) error {
	// 序列化消息
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	// 检查连接是否已关闭
	c.closedMutex.RLock()
	closed := c.closed
	c.closedMutex.RUnlock()
	if closed {
		return fmt.Errorf("连接已关闭")
	}

	// 发送消息
	select {
	case c.writeChan <- data:
		return nil
	default:
		return fmt.Errorf("写入通道已满")
	}
}

// SendErrorMessage 发送错误消息
func (c *Client) SendErrorMessage(errMsg string, details string) error {
	msg := ErrorMessage{
		BaseMessage: BaseMessage{Type: MsgError},
		Message:     errMsg,
		Details:     details,
	}
	return c.SendMessage(msg)
}

// Close 关闭客户端连接
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		// 设置关闭标志
		c.closedMutex.Lock()
		c.closed = true
		c.closedMutex.Unlock()

		// 取消上下文
		c.readCancel()
		c.writeCancel()

		// 关闭通道
		close(c.writeChan)

		// 从Hub注销
		if c.username != "" {
			c.hub.UnregisterClient(c.username)
		}

		// 关闭连接
		c.conn.Close()

		log.Printf("客户端连接已关闭: %s", c.username)
	})
}
