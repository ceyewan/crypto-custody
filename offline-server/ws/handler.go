package ws

import (
	"offline-server/clog"
	"offline-server/manager"
	"offline-server/storage"
	mem_storage "offline-server/ws/storage"
)

// MessageHandler 消息处理器
// 负责处理各种WebSocket消息
type MessageHandler struct {
	shareStorage      storage.IShareStorage
	seStorage         storage.ISeStorage
	offlineKeyStorage storage.IOfflineKeyStorage
	keyGenStorage     storage.IKeyGenStorage
	signStorage       storage.ISignStorage
	auditStorage      storage.IAuditStorage
	sessionManager    *mem_storage.SessionManager
	managerRuntime    manager.SessionRuntime

	// 拆分后的消息处理器
	keygenHandler  *KeyGenHandler  // 密钥生成消息处理器
	signHandler    *SignHandler    // 签名消息处理器
	destroyHandler *DestroyHandler // 密钥销毁消息处理器
}

// NewMessageHandler 创建新的消息处理器
func NewMessageHandler() *MessageHandler {
	shareStorage := storage.GetShareStorage()
	seStorage := storage.GetSeStorage()
	offlineKeyStorage := storage.GetOfflineKeyStorage()
	keyGenStorage := storage.GetKeyGenStorage()
	signStorage := storage.GetSignStorage()
	auditStorage := storage.GetAuditStorage()
	sessionManager := mem_storage.GetSessionManager()
	managerRuntime := manager.NewSessionRuntimeFromEnv()

	handler := &MessageHandler{
		shareStorage:      shareStorage,
		seStorage:         seStorage,
		offlineKeyStorage: offlineKeyStorage,
		keyGenStorage:     keyGenStorage,
		signStorage:       signStorage,
		auditStorage:      auditStorage,
		sessionManager:    sessionManager,
		managerRuntime:    managerRuntime,
	}

	// 创建子处理器
	handler.keygenHandler = NewKeyGenHandler(shareStorage, seStorage, offlineKeyStorage, keyGenStorage, auditStorage, sessionManager, managerRuntime)
	handler.signHandler = NewSignHandler(shareStorage, seStorage, offlineKeyStorage, signStorage, auditStorage, sessionManager, managerRuntime)
	handler.destroyHandler = NewDestroyHandler(shareStorage, seStorage, offlineKeyStorage, auditStorage, sessionManager)

	clog.Debug("创建消息处理器实例")
	return handler
}

// Close 关闭消息处理器持有的会话级资源。
func (h *MessageHandler) Close() error {
	if h.managerRuntime == nil {
		return nil
	}
	return h.managerRuntime.StopAll()
}
