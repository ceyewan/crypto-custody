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

// Server WebSocket服务器
// 管理WebSocket连接、客户端、会话等
type Server struct {
	// 服务器配置
	addr    string          // 监听地址
	server  *http.Server    // HTTP服务器
	hub     *Hub            // 客户端集线器
	handler *MessageHandler // 消息处理器

	// 状态管理
	started bool // 服务器是否已启动
}

// NewServer 创建新的WebSocket服务器
func NewServer(addr string) *Server {
	// 创建消息处理器
	handler := NewMessageHandler()

	// 创建客户端集线器
	hub := NewHub(handler)

	// 创建服务器
	server := &Server{
		addr:    addr,
		handler: handler,
		hub:     hub,
	}

	clog.Debug("创建WebSocket服务器",
		clog.String("addr", addr))

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
		clog.String("handler_path", "/ws"))

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

// handleWebSocket 处理WebSocket连接请求
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 升级HTTP连接为WebSocket连接
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		clog.Error("升级WebSocket连接失败", clog.Err(err),
			clog.String("remote_addr", r.RemoteAddr))
		return
	}

	// 创建客户端
	client := NewClient(conn, s.hub, s.handler)

	// 启动客户端
	client.Start()

	clog.Info("新的WebSocket连接已建立",
		clog.String("remote_addr", r.RemoteAddr))
	clog.Debug("新的WebSocket连接详情",
		clog.String("remote_addr", r.RemoteAddr),
		clog.String("user_agent", r.UserAgent()),
		clog.String("protocol", r.Proto))
}
