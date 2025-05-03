package ws

import (
	"offline-server/clog"
	"offline-server/storage"
	mem_storage "offline-server/ws/storage"
)

// MessageHandler 消息处理器
// 负责处理各种WebSocket消息
type MessageHandler struct {
	shareStorage   storage.IShareStorage       // 私钥分片存储接口
	seStorage      storage.ISeStorage          // 安全芯片存储接口
	sessionManager *mem_storage.SessionManager // 会话管理器

	// 拆分后的消息处理器
	keygenHandler *KeyGenHandler // 密钥生成消息处理器
	signHandler   *SignHandler   // 签名消息处理器
}

// NewMessageHandler 创建新的消息处理器
func NewMessageHandler() *MessageHandler {
	shareStorage := storage.GetShareStorage()
	seStorage := storage.GetSeStorage()
	sessionManager := mem_storage.GetSessionManager()

	handler := &MessageHandler{
		shareStorage:   shareStorage,
		seStorage:      seStorage,
		sessionManager: sessionManager,
	}

	// 创建子处理器
	handler.keygenHandler = NewKeyGenHandler(shareStorage, seStorage, sessionManager)
	handler.signHandler = NewSignHandler(shareStorage, seStorage, sessionManager)

	clog.Debug("创建消息处理器实例")
	return handler
}
