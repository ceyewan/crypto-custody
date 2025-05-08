package ws

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"offline-server/clog"

	"github.com/gorilla/websocket"
)

// WebSocket连接升级器配置
var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096, // 读缓冲区大小
	WriteBufferSize: 4096, // 写缓冲区大小
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许所有源的连接，生产环境应限制
	},
}

// ServerConfig 服务器配置
type ServerConfig struct {
	PingInterval     time.Duration // ping发送间隔
	ReadTimeout      time.Duration // 读超时时间
	WriteTimeout     time.Duration // 写超时时间
	MessageSizeLimit int64         // 消息大小限制
}

// DefaultServerConfig 默认服务器配置
var DefaultServerConfig = ServerConfig{
	PingInterval:     30 * time.Second,
	ReadTimeout:      60 * time.Second,
	WriteTimeout:     10 * time.Second,
	MessageSizeLimit: 65536, // 64KB
}

// Server WebSocket服务器
// 管理WebSocket连接、客户端、会话等
type Server struct {
	// 服务器配置
	addr    string          // 监听地址
	config  ServerConfig    // 服务器配置
	server  *http.Server    // HTTP服务器
	hub     *Hub            // 客户端集线器
	handler *MessageHandler // 消息处理器

	// 状态管理
	started bool        // 服务器是否已启动
	stats   serverStats // 服务器统计信息
}

// serverStats 服务器统计信息
type serverStats struct {
	totalConnections   int       // 总连接数
	activeConnections  int       // 活动连接数
	failedConnections  int       // 失败连接数
	lastConnectionTime time.Time // 最后连接时间
	startTime          time.Time // 服务启动时间
}

// NewServer 创建新的WebSocket服务器
func NewServer(addr string) *Server {
	return NewServerWithConfig(addr, DefaultServerConfig)
}

// NewServerWithConfig 使用自定义配置创建新的WebSocket服务器
func NewServerWithConfig(addr string, config ServerConfig) *Server {
	// 创建消息处理器
	handler := NewMessageHandler()

	// 创建客户端集线器
	hub := NewHub(handler)

	// 创建服务器
	server := &Server{
		addr:    addr,
		config:  config,
		handler: handler,
		hub:     hub,
		stats: serverStats{
			startTime: time.Now(),
		},
	}

	clog.Debug("创建WebSocket服务器",
		clog.String("addr", addr),
		clog.Duration("ping_interval", config.PingInterval),
		clog.Duration("read_timeout", config.ReadTimeout),
		clog.Duration("write_timeout", config.WriteTimeout),
		clog.Int64("message_size_limit", config.MessageSizeLimit))

	return server
}

// Start 启动WebSocket服务器
func (s *Server) Start() error {
	if s.started {
		return fmt.Errorf("服务器已启动")
	}

	// 创建HTTP服务器
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: mux,
		// 设置超时
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
	}

	// 启动HTTP服务器
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("监听地址失败: %w", err)
	}

	s.started = true
	clog.Info("WebSocket服务器已启动",
		clog.String("addr", s.addr))
	clog.Debug("WebSocket服务器启动详情",
		clog.String("addr", s.addr),
		clog.String("handler_path", "/ws"),
		clog.Time("start_time", s.stats.startTime))

	// 在新协程中启动服务器
	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			clog.Error("HTTP服务器错误", clog.Err(err))
		}
	}()

	return nil
}

// Stop 停止WebSocket服务器
func (s *Server) Stop() error {
	if !s.started {
		return nil
	}

	// 创建上下文，设置超时
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 关闭HTTP服务器
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("关闭HTTP服务器失败: %w", err)
	}

	s.started = false
	clog.Info("WebSocket服务器已停止")
	clog.Debug("WebSocket服务器关闭详情",
		clog.String("addr", s.addr),
		clog.Duration("timeout", 5*time.Second))

	return nil
}

// GetConnectionStats 获取连接统计信息
func (s *Server) GetConnectionStats() (int, int, int) {
	totalConns, reconnections, _ := s.hub.GetConnectionStats()
	return totalConns, s.stats.failedConnections, reconnections
}

// handleWebSocket 处理WebSocket连接请求
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 记录连接时间
	s.stats.lastConnectionTime = time.Now()
	s.stats.totalConnections++

	// 获取真实客户端IP
	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}

	clog.Debug("收到WebSocket连接请求",
		clog.String("remote_addr", clientIP),
		clog.String("user_agent", r.UserAgent()))

	// 升级HTTP连接为WebSocket连接
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.stats.failedConnections++
		clog.Error("升级WebSocket连接失败", clog.Err(err),
			clog.String("remote_addr", clientIP))
		return
	}

	// 设置连接参数
	conn.SetReadLimit(s.config.MessageSizeLimit)

	// 创建客户端
	client := NewClient(conn, s.hub, s.handler)

	// 启动客户端
	client.Start()
	s.stats.activeConnections++

	clog.Info("新的WebSocket连接已建立",
		clog.String("remote_addr", clientIP))
	clog.Debug("新的WebSocket连接详情",
		clog.String("remote_addr", clientIP),
		clog.String("user_agent", r.UserAgent()),
		clog.String("protocol", r.Proto),
		clog.Int("active_connections", s.stats.activeConnections),
		clog.Int("total_connections", s.stats.totalConnections))
}
