package ws

import (
	"fmt"
	"sync"
	"time"

	"offline-server/clog"
)

// Hub 客户端集线器
// 负责管理所有客户端连接，提供注册、消息分发等功能
type Hub struct {
	// 客户端管理
	clients map[string]*Client // 用户名 -> 客户端
	mutex   sync.RWMutex       // 保护映射的互斥锁

	// 消息处理
	handler *MessageHandler // 消息处理器

	// 连接状态统计
	connectionStats struct {
		totalConnections      int       // 当前连接总数
		reconnections         int       // 重连次数
		lastDisconnectionTime time.Time // 最后一次断开连接的时间
		mutex                 sync.RWMutex
	}
}

// NewHub 创建新的客户端集线器
func NewHub(handler *MessageHandler) *Hub {
	hub := &Hub{
		clients: make(map[string]*Client),
		handler: handler,
	}

	// 初始化连接统计
	hub.connectionStats.lastDisconnectionTime = time.Now()

	// 启动定期清理无效连接
	go hub.periodicConnectionCheck()

	return hub
}

// periodicConnectionCheck 定期检查并清理无效连接
func (h *Hub) periodicConnectionCheck() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		h.cleanupDeadConnections()
	}
}

// cleanupDeadConnections 清理无效连接
func (h *Hub) cleanupDeadConnections() {
	h.mutex.RLock()
	clientsCopy := make(map[string]*Client)
	for username, client := range h.clients {
		clientsCopy[username] = client
	}
	h.mutex.RUnlock()

	deadClients := 0

	// 遍历所有客户端，检查连接状态
	for username, client := range clientsCopy {
		// 检查连接是否已关闭
		client.closedMutex.RLock()
		closed := client.closed
		client.closedMutex.RUnlock()

		if closed {
			// 从Hub注销
			h.UnregisterClient(username)
			deadClients++
			continue
		}
	}

	if deadClients > 0 {
		clog.Info("清理无效连接",
			clog.Int("dead_connections", deadClients),
			clog.Int("remaining_connections", len(h.clients)))
	}
}

// RegisterClient 注册客户端
func (h *Hub) RegisterClient(username string, client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// 检查是否有同名客户端已存在
	if oldClient, exists := h.clients[username]; exists {
		// 如果存在，先关闭旧连接
		clog.Info("用户连接已存在，关闭旧连接",
			clog.String("username", username),
			clog.String("role", string(oldClient.Role())))

		// 更新重连统计
		h.connectionStats.mutex.Lock()
		h.connectionStats.reconnections++
		h.connectionStats.mutex.Unlock()

		// 关闭旧连接，但要确保不会在持有Hub锁的情况下调用，避免死锁
		go oldClient.Close()
	} else {
		// 更新连接总数
		h.connectionStats.mutex.Lock()
		h.connectionStats.totalConnections++
		h.connectionStats.mutex.Unlock()
	}

	// 添加客户端
	h.clients[username] = client

	clog.Info("注册客户端",
		clog.String("username", username),
		clog.String("role", string(client.Role())))
	clog.Debug("客户端注册详情",
		clog.String("username", username),
		clog.String("role", string(client.Role())),
		clog.Int("total_clients", len(h.clients)))
}

// UnregisterClient 注销客户端
func (h *Hub) UnregisterClient(username string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	// 检查客户端是否存在
	_, exists := h.clients[username]
	if !exists {
		// 客户端已不存在，可能已被其他操作注销
		return
	}

	// 更新连接统计
	h.connectionStats.mutex.Lock()
	h.connectionStats.totalConnections--
	h.connectionStats.lastDisconnectionTime = time.Now()
	h.connectionStats.mutex.Unlock()

	// 删除客户端
	delete(h.clients, username)

	clog.Info("注销客户端",
		clog.String("username", username))
	clog.Debug("客户端注销详情",
		clog.String("username", username),
		clog.Int("remaining_clients", len(h.clients)))
}

// GetClient 获取客户端
func (h *Hub) GetClient(username string) (*Client, bool) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	client, exists := h.clients[username]
	return client, exists
}

// GetAllClients 获取所有客户端
func (h *Hub) GetAllClients() map[string]*Client {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	// 创建副本
	clients := make(map[string]*Client)
	for k, v := range h.clients {
		clients[k] = v
	}
	return clients
}

// GetConnectionStats 获取连接统计信息
func (h *Hub) GetConnectionStats() (int, int, time.Time) {
	h.connectionStats.mutex.RLock()
	defer h.connectionStats.mutex.RUnlock()

	return h.connectionStats.totalConnections,
		h.connectionStats.reconnections,
		h.connectionStats.lastDisconnectionTime
}

// SendMessageToUser 向指定用户发送消息
func (h *Hub) SendMessageToUser(username string, msg Message) error {
	// 获取客户端
	client, exists := h.GetClient(username)
	if !exists {
		return fmt.Errorf("用户 %s 不在线", username)
	}

	// 发送消息
	return client.SendMessage(msg)
}

// BroadcastMessage 广播消息给所有客户端
func (h *Hub) BroadcastMessage(msg Message) {
	clients := h.GetAllClients()
	clog.Debug("开始广播消息",
		clog.String("msg_type", string(msg.GetType())),
		clog.Int("client_count", len(clients)))

	for username, client := range clients {
		if err := client.SendMessage(msg); err != nil {
			clog.Error("向用户广播消息失败",
				clog.String("username", username),
				clog.Err(err))
		}
	}
}

// BroadcastMessageToRole 广播消息给指定角色的客户端
func (h *Hub) BroadcastMessageToRole(msg Message, role ClientRole) {
	clients := h.GetAllClients()
	targetCount := 0

	for username, client := range clients {
		if client.Role() == role {
			targetCount++
			if err := client.SendMessage(msg); err != nil {
				clog.Error("向指定角色用户广播消息失败",
					clog.String("username", username),
					clog.String("role", string(role)),
					clog.Err(err))
			}
		}
	}

	clog.Debug("向指定角色广播消息完成",
		clog.String("role", string(role)),
		clog.String("msg_type", string(msg.GetType())),
		clog.Int("target_count", targetCount))
}
