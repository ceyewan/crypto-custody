package main

import (
	"context"
	"sync"

	"offline-client-wails/mpc_core/clog"
	"offline-client-wails/mpc_core/config"
	"offline-client-wails/mpc_core/controllers"
	"offline-client-wails/mpc_core/services"
)

// KeyGenerationRequest 封装了密钥生成所需的参数
type KeyGenerationRequest struct {
	Threshold int    `json:"threshold"`
	Parties   int    `json:"parties"`
	Index     int    `json:"index"`
	UserName  string `json:"user_name"`
}

// SignMessageRequest 封装了消息签名所需的参数
type SignMessageRequest struct {
	Message      string `json:"message"`
	Parties      string `json:"parties"`
	UserName     string `json:"user_name"`
	Address      string `json:"address"`
	EncryptedKey []byte `json:"encrypted_key"`
	Signature    []byte `json:"signature"` // 用于授权安全芯片操作的签名
}

// DeleteMessageRequest 封装了删除消息（密钥）所需的参数
type DeleteMessageRequest struct {
	UserName  string `json:"user_name"`
	Address   string `json:"address"`
	Signature []byte `json:"signature"` // 用于授权安全芯片操作的签名
}

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
func (ws *WailsServices) PerformKeyGeneration(req KeyGenerationRequest) (interface{}, error) {
	if !ws.initialized {
		if err := ws.Init(); err != nil {
			return nil, err
		}
	}

	ctx := context.Background()
	// 使用从前端请求传递的参数
	address, encryptedKey, err := ws.mpcService.KeyGeneration(ctx, req.Threshold, req.Parties, req.Index, "keygen_temp.json", req.UserName)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"address":       address,
		"encrypted_key": encryptedKey,
	}, nil
}

// PerformSignMessage 执行消息签名
func (ws *WailsServices) PerformSignMessage(req SignMessageRequest) (interface{}, error) {
	if !ws.initialized {
		if err := ws.Init(); err != nil {
			return nil, err
		}
	}

	ctx := context.Background()
	// 使用从前端请求传递的参数
	result, err := ws.mpcService.SignMessage(
		ctx,
		req.Parties,
		req.Message,
		"sign_temp.json", // 临时文件名仍然可以在后端定义
		req.UserName,
		req.Address,
		req.EncryptedKey,
		req.Signature,
	)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"signature": result,
		"message":   req.Message,
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

// PerformDeleteMessage 从安全芯片中删除一个密钥记录
func (ws *WailsServices) PerformDeleteMessage(req DeleteMessageRequest) error {
	if !ws.initialized {
		if err := ws.Init(); err != nil {
			return err
		}
	}

	// 调用 securityService 来执行删除
	return ws.securityService.DeleteData(req.UserName, req.Address, req.Signature)
}
