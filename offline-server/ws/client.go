package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"offline-server/clog"
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

	clog.Debug("创建了新的客户端",
		clog.String("remote_addr", conn.RemoteAddr().String()),
		clog.Int("read_buffer", ReadBufferSize),
		clog.Int("write_buffer", WriteBufferSize))

	return client
}

// Start 启动客户端消息处理循环
func (c *Client) Start() {
	// 启动读写协程
	c.wg.Add(2)
	go c.readPump()
	go c.writePump()

	clog.Debug("启动客户端处理协程",
		clog.String("remote_addr", c.conn.RemoteAddr().String()))
}

// GetUserName 获取客户端用户名
func (c *Client) GetUserName() string {
	return c.username
}

// SetUserName 设置客户端用户名
func (c *Client) SetUserName(username string) {
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

	clog.Debug("启动读取消息循环",
		clog.String("username", c.username),
		clog.String("remote_addr", c.conn.RemoteAddr().String()),
		clog.Int("max_message_size", MaxMessageSize),
		clog.Duration("pong_wait", PongWait))

	// 持续读取消息
	for {
		select {
		case <-c.readCtx.Done():
			// 上下文已取消，退出协程
			clog.Debug("读取消息循环退出 - 上下文取消",
				clog.String("username", c.username))
			return
		default:
			// 读取消息
			_, message, err := c.conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err,
					websocket.CloseGoingAway,
					websocket.CloseAbnormalClosure) {
					clog.Error("WebSocket读取错误",
						clog.Err(err),
						clog.String("username", c.username))
				} else {
					clog.Debug("WebSocket连接关闭",
						clog.String("username", c.username),
						clog.String("reason", err.Error()))
				}
				return
			}

			// 解析消息类型用于日志记录
			var baseMsg BaseMessage
			if err := json.Unmarshal(message, &baseMsg); err == nil {
				clog.Debug("收到WebSocket消息",
					clog.String("username", c.username),
					clog.String("msg_type", string(baseMsg.Type)),
					clog.Int("msg_size", len(message)))
			}

			// 处理收到的消息
			if err := c.handleMessage(message); err != nil {
				clog.Error("处理客户端消息失败",
					clog.Err(err),
					clog.String("username", c.username))

				// 发送错误响应
				if errMsg := c.SendErrorMessage(err.Error(), ""); errMsg != nil {
					clog.Error("发送错误消息失败",
						clog.Err(errMsg),
						clog.String("username", c.username))
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

	clog.Debug("启动写入消息循环",
		clog.String("username", c.username),
		clog.String("remote_addr", c.conn.RemoteAddr().String()),
		clog.Duration("ping_period", PingPeriod),
		clog.Duration("write_wait", WriteWait))

	for {
		select {
		case <-c.writeCtx.Done():
			// 上下文已取消，退出协程
			clog.Debug("写入消息循环退出 - 上下文取消",
				clog.String("username", c.username))
			return
		case message, ok := <-c.writeChan:
			// 设置写入截止时间
			c.conn.SetWriteDeadline(time.Now().Add(WriteWait))
			if !ok {
				// 通道已关闭
				clog.Debug("写入通道已关闭，发送关闭消息",
					clog.String("username", c.username))
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 解析消息类型用于日志记录
			var baseMsg BaseMessage
			if err := json.Unmarshal(message, &baseMsg); err == nil {
				clog.Debug("发送WebSocket消息",
					clog.String("username", c.username),
					clog.String("msg_type", string(baseMsg.Type)),
					clog.Int("msg_size", len(message)))
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				clog.Error("获取WebSocket消息写入器失败",
					clog.Err(err),
					clog.String("username", c.username))
				return
			}
			w.Write(message)

			// 添加队列中的所有消息
			n := len(c.writeChan)
			if n > 0 {
				clog.Debug("写入通道有额外待发送消息",
					clog.String("username", c.username),
					clog.Int("additional_messages", n))
			}

			// 重要：不再一次性发送多条消息，避免JSON解析错误
			// 之前的实现会导致多条JSON消息连在一起无法解析
			if n > 0 {
				clog.Warn("写入通道有多条待发消息，但已禁用批量发送以避免JSON解析错误",
					clog.String("username", c.username),
					clog.Int("queued_messages", n))
			}
			// 不再处理队列中的其他消息，每条消息单独发送
			// for i := 0; i < n; i++ {
			//     w.Write(<-c.writeChan)
			// }

			if err := w.Close(); err != nil {
				clog.Error("关闭WebSocket消息写入器失败",
					clog.Err(err),
					clog.String("username", c.username))
				return
			}
		case <-ticker.C:
			// 设置写入截止时间
			c.conn.SetWriteDeadline(time.Now().Add(WriteWait))

			// 发送ping消息以保持连接
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				clog.Error("发送ping消息失败",
					clog.Err(err),
					clog.String("username", c.username))
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

		clog.Debug("处理注册消息",
			clog.String("username", registerMsg.Username),
			clog.String("role", string(registerMsg.Role)))

		return c.handleRegisterMessage(registerMsg)

	// 密钥生成相关消息
	case MsgKeyGenRequest, MsgKeyGenResponse, MsgKeyGenResult:
		// 非注册消息 - 检查客户端是否已注册
		if c.username == "" {
			return fmt.Errorf("客户端尚未注册，请先发送注册消息")
		}

		clog.Debug("转发密钥生成消息到处理器",
			clog.String("username", c.username),
			clog.String("msg_type", string(baseMsg.Type)))

		return c.handler.keygenHandler.ProcessMessage(baseMsg.Type, message, c)

	// 签名相关消息
	case MsgSignRequest, MsgSignResponse, MsgSignResult:
		// 非注册消息 - 检查客户端是否已注册
		if c.username == "" {
			return fmt.Errorf("客户端尚未注册，请先发送注册消息")
		}

		clog.Debug("转发签名消息到处理器",
			clog.String("username", c.username),
			clog.String("msg_type", string(baseMsg.Type)))

		return c.handler.signHandler.ProcessMessage(baseMsg.Type, message, c)

	default:
		if c.username == "" {
			return fmt.Errorf("客户端尚未注册，请先发送注册消息")
		}

		clog.Error("不支持的消息类型",
			clog.String("msg_type", string(baseMsg.Type)),
			clog.String("username", c.username))
		return fmt.Errorf("不支持的消息类型: %s", baseMsg.Type)
	}
}

// handleRegisterMessage 处理注册消息
func (c *Client) handleRegisterMessage(msg RegisterMessage) error {
	// 格式化Token: 移除Bearer前缀
	tokenString := msg.Token
	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}

	// 验证Token
	username, role, err := tools.ValidateToken(tokenString)
	if err != nil {
		clog.Error("验证Token失败",
			clog.Err(err),
			clog.String("claimed_username", msg.Username))
		return fmt.Errorf("验证Token失败: %w", err)
	}

	// 验证Token中的信息与消息中的信息是否一致
	if username != msg.Username {
		clog.Error("Token用户名不匹配",
			clog.String("token_username", username),
			clog.String("msg_username", msg.Username))
		return fmt.Errorf("token中的用户名与消息中的用户名不匹配")
	}
	if ClientRole(role) != msg.Role {
		clog.Error("Token角色不匹配",
			clog.String("token_role", role),
			clog.String("msg_role", string(msg.Role)))
		return fmt.Errorf("token中的角色与消息中的角色不匹配")
	}

	// 设置客户端属性
	c.SetUserName(msg.Username)
	c.SetRole(msg.Role)

	// 注册到Hub
	c.hub.RegisterClient(msg.Username, c)

	clog.Info("客户端注册成功",
		clog.String("username", msg.Username),
		clog.String("role", string(msg.Role)),
		clog.String("remote_addr", c.conn.RemoteAddr().String()))

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

	clog.Debug("准备发送消息",
		clog.String("username", c.username),
		clog.String("msg_type", string(msg.GetType())),
		clog.Int("data_size", len(data)))

	// 发送消息
	select {
	case c.writeChan <- data:
		return nil
	default:
		clog.Error("发送消息失败，写入通道已满",
			clog.String("username", c.username),
			clog.String("msg_type", string(msg.GetType())))
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

	clog.Debug("发送错误消息",
		clog.String("username", c.username),
		clog.String("error_msg", errMsg),
		clog.String("details", details))

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

		clog.Info("客户端连接已关闭",
			clog.String("username", c.username),
			clog.String("remote_addr", c.conn.RemoteAddr().String()))
	})
}
