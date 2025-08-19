package main

import (
	"context"
	"sync"

	"offline-client-wails/mpc_core/clog"
	"offline-client-wails/mpc_core/config"
	"offline-client-wails/mpc_core/controllers"
	"offline-client-wails/mpc_core/services"
)

// WailsServices 包装原有的服务层，为 Wails 前端提供统一接口
type WailsServices struct {
	cfg             *config.Config
	securityService *services.SecurityService
	mpcService      *services.MPCService
	initOnce        sync.Once
	initialized     bool
}

var wailsServices *WailsServices

// GetWailsServices 获取单例 WailsServices 实例
func GetWailsServices() *WailsServices {
	if wailsServices == nil {
		wailsServices = &WailsServices{}
	}
	return wailsServices
}

// Init 初始化服务
func (ws *WailsServices) Init() error {
	var initErr error

	ws.initOnce.Do(func() {
		var err error

		// 加载配置
		ws.cfg, err = config.LoadConfig()
		if err != nil {
			clog.Error("加载配置失败", clog.String("error", err.Error()))
			initErr = err
			return
		}

		// 创建安全芯片服务
		ws.securityService, err = services.NewSecurityService(ws.cfg)
		if err != nil {
			clog.Error("创建安全芯片服务失败", clog.String("error", err.Error()))
			initErr = err
			return
		}

		// 创建MPC服务
		ws.mpcService = services.NewMPCService(ws.cfg, ws.securityService)

		// 初始化控制器
		if err := controllers.Init(); err != nil {
			clog.Error("初始化控制器失败", clog.String("error", err.Error()))
			initErr = err
			return
		}

		ws.initialized = true
		clog.Info("WailsServices初始化成功")
	})

	return initErr
}

// PerformKeyGeneration 执行密钥生成
func (ws *WailsServices) PerformKeyGeneration() (interface{}, error) {
	if !ws.initialized {
		if err := ws.Init(); err != nil {
			return nil, err
		}
	}

	ctx := context.Background()
	// 使用默认参数：threshold=2, parties=3, index=1
	address, encryptedKey, err := ws.mpcService.KeyGeneration(ctx, 2, 3, 1, "keygen_temp.json", "default_user")
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"address":       address,
		"encrypted_key": encryptedKey,
	}, nil
}

// PerformSignMessage 执行消息签名
func (ws *WailsServices) PerformSignMessage(message string) (interface{}, error) {
	if !ws.initialized {
		if err := ws.Init(); err != nil {
			return nil, err
		}
	}

	ctx := context.Background()
	// 使用默认参数进行签名
	// 注意：在实际使用中，这些参数应该从存储中获取或由用户提供
	parties := "3"
	filename := "sign_temp.json"
	userName := "default_user"
	address := ""            // 需要从之前的密钥生成中获取
	encryptedKey := []byte{} // 需要从安全芯片中获取
	signature := []byte{}    // 初始为空

	result, err := ws.mpcService.SignMessage(ctx, parties, message, filename, userName, address, encryptedKey, signature)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"signature": result,
		"message":   message,
	}, nil
}

// GetCPLCInfo 获取CPLC信息
func (ws *WailsServices) GetCPLCInfo() (interface{}, error) {
	if !ws.initialized {
		if err := ws.Init(); err != nil {
			return nil, err
		}
	}

	cplcInfo, err := ws.securityService.GetCPLC()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"cplc_info": cplcInfo,
	}, nil
}

// PerformDeleteMessage 执行删除消息
func (ws *WailsServices) PerformDeleteMessage() error {
	if !ws.initialized {
		if err := ws.Init(); err != nil {
			return err
		}
	}

	// 这里可以添加具体的删除逻辑
	// 现在先返回成功
	clog.Info("执行删除操作")
	return nil
}
