package ws

import (
	"fmt"
	"sync"

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
}

// NewHub 创建新的客户端集线器
func NewHub(handler *MessageHandler) *Hub {
	return &Hub{
		clients: make(map[string]*Client),
		handler: handler,
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
		oldClient.Close()
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
