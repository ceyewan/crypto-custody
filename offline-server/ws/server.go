package ws

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

var (
	// upgrader 用于将HTTP连接升级到WebSocket协议
	// 在生产环境中应该限制允许的来源
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // 允许所有源的连接，在生产环境中应该限制
		},
	}
)

// Server 表示WebSocket服务器，管理WebSocket连接并处理消息
type Server struct {
	store      Storage         // 存储客户端连接和其他数据
	managerCmd *exec.Cmd       // 管理外部进程
	handler    *MessageHandler // 消息处理器
	server     *http.Server    // HTTP服务器实例
}

// NewServer 创建并初始化一个新的WebSocket服务器实例
// 返回准备好的服务器，但尚未启动
func NewServer() *Server {
	store := NewMemoryStorage()
	handler := NewMessageHandler(store)

	return &Server{
		store:   store,
		handler: handler,
	}
}

// Start 启动WebSocket服务器，包括初始化外部Manager进程和HTTP服务
// 可以指定监听端口，如果端口为0，则自动选择一个可用端口
// 返回启动过程中可能发生的错误
func (s *Server) Start(port int) error {
	// 启动Manager
	var err error
	s.managerCmd, err = RunManager()
	if err != nil {
		return fmt.Errorf("启动Manager失败: %v", err)
	}
	log.Println("Manager已启动")

	// 设置WebSocket处理程序
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleConnection)

	// 获取监听地址
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("无法监听端口 %d: %v", port, err)
	}

	// 创建HTTP服务器
	s.server = &http.Server{
		Handler: mux,
	}

	// 获取实际使用的端口
	actualPort := listener.Addr().(*net.TCPAddr).Port
	log.Printf("服务器启动在 :%d 端口", actualPort)

	// 启动HTTP服务器
	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP服务器启动失败: %v", err)
		}
	}()

	return nil
}

// Stop 优雅地停止WebSocket服务器，关闭所有连接和外部进程
// 给予连接一定的时间完成关闭操作
func (s *Server) Stop() {
	// 关闭所有客户端连接
	clients := s.store.GetAllClients()
	for _, conn := range clients {
		if wsConn, ok := conn.(*websocket.Conn); ok {
			wsConn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, "服务器关闭"))
			wsConn.Close()
		}
	}

	// 优雅地关闭HTTP服务器
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.server.Shutdown(ctx); err != nil {
			log.Printf("HTTP服务器关闭出错: %v", err)
		}
	}

	// 终止Manager进程
	if s.managerCmd != nil && s.managerCmd.Process != nil {
		if err := s.managerCmd.Process.Signal(syscall.SIGTERM); err != nil {
			log.Printf("无法正常终止Manager: %v", err)
			s.managerCmd.Process.Kill()
		}
		log.Println("Manager已终止")
	}

	log.Println("服务器已关闭")
}

// handleConnection 处理新的WebSocket连接请求
// 负责升级HTTP连接到WebSocket并初始化客户端实例
func (s *Server) handleConnection(w http.ResponseWriter, r *http.Request) {
	// 升级HTTP连接为WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("升级连接失败: %v", err)
		return
	}

	// 设置连接处理
	client := NewClient(conn, s.handler, s.store)

	// 启动客户端处理
	go client.Listen()
}
