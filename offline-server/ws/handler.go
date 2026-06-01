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
	approvalStore     storage.IApprovalStorage
	sessionManager    *mem_storage.SessionManager
	managerRuntime    manager.SessionRuntime

	// 拆分后的消息处理器
	keygenHandler   *KeyGenHandler   // 密钥生成消息处理器
	signHandler     *SignHandler     // 签名消息处理器
	destroyHandler  *DestroyHandler  // 密钥销毁消息处理器
	transferHandler *TransferHandler // 分片移交消息处理器
}

// NewMessageHandler 创建新的消息处理器
func NewMessageHandler() *MessageHandler {
	shareStorage := storage.GetShareStorage()
	seStorage := storage.GetSeStorage()
	offlineKeyStorage := storage.GetOfflineKeyStorage()
	keyGenStorage := storage.GetKeyGenStorage()
	signStorage := storage.GetSignStorage()
	auditStorage := storage.GetAuditStorage()
	approvalStore := storage.GetApprovalStorage()
	sessionManager := mem_storage.GetSessionManager()
	managerRuntime := manager.NewSessionRuntimeFromEnv()

	return NewMessageHandlerWithDependencies(
		shareStorage,
		seStorage,
		offlineKeyStorage,
		keyGenStorage,
		signStorage,
		auditStorage,
		approvalStore,
		sessionManager,
		managerRuntime,
	)
}

// NewMessageHandlerWithDependencies 创建使用显式依赖的消息处理器，主要用于
// 本地 smoke、测试和外部组装场景。生产默认路径仍使用 NewMessageHandler。
func NewMessageHandlerWithDependencies(
	shareStorage storage.IShareStorage,
	seStorage storage.ISeStorage,
	offlineKeyStorage storage.IOfflineKeyStorage,
	keyGenStorage storage.IKeyGenStorage,
	signStorage storage.ISignStorage,
	auditStorage storage.IAuditStorage,
	approvalStore storage.IApprovalStorage,
	sessionManager *mem_storage.SessionManager,
	managerRuntime manager.SessionRuntime,
) *MessageHandler {
	if sessionManager == nil {
		sessionManager = mem_storage.NewSessionManager()
	}
	if managerRuntime == nil {
		managerRuntime = manager.NewSessionRuntimeFromEnv()
	}

	handler := &MessageHandler{
		shareStorage:      shareStorage,
		seStorage:         seStorage,
		offlineKeyStorage: offlineKeyStorage,
		keyGenStorage:     keyGenStorage,
		signStorage:       signStorage,
		auditStorage:      auditStorage,
		approvalStore:     approvalStore,
		sessionManager:    sessionManager,
		managerRuntime:    managerRuntime,
	}

	// 创建子处理器
	handler.keygenHandler = NewKeyGenHandler(shareStorage, seStorage, offlineKeyStorage, keyGenStorage, auditStorage, sessionManager, managerRuntime)
	handler.signHandler = NewSignHandler(shareStorage, seStorage, offlineKeyStorage, signStorage, auditStorage, sessionManager, managerRuntime)
	handler.destroyHandler = NewDestroyHandler(shareStorage, seStorage, offlineKeyStorage, auditStorage, approvalStore, sessionManager)
	handler.transferHandler = NewTransferHandler(shareStorage, auditStorage, approvalStore, sessionManager)

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
