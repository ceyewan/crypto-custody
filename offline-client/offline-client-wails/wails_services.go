package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"

	"offline-client-wails/mpc_core/clog"
	"offline-client-wails/mpc_core/config"
	"offline-client-wails/mpc_core/models"
	"offline-client-wails/mpc_core/services"
)

// 复用原有的 models，无需重新定义

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

		ws.initialized = true
		clog.Info("WailsServices初始化成功")
	})

	return initErr
}

// PerformKeyGeneration 执行密钥生成
func (ws *WailsServices) PerformKeyGeneration(req models.KeyGenRequest) (interface{}, error) {
	if !ws.initialized {
		if err := ws.Init(); err != nil {
			return nil, err
		}
	}

	ctx := context.Background()
	// 使用从前端请求传递的参数，使用 models 中定义的字段
	address, encryptedKey, err := ws.mpcService.KeyGeneration(ctx, req.Threshold, req.Parties, req.Index, req.Filename, req.UserName)
	if err != nil {
		return nil, err
	}

	// 确保地址格式正确（添加0x前缀如果没有）
	if !strings.HasPrefix(address, "0x") {
		address = "0x" + address
	}

	// 转换加密密钥为Base64字符串（仿照原controller逻辑）
	encryptedKeyBase64 := base64.StdEncoding.EncodeToString(encryptedKey)

	// 返回与原有 models.KeyGenResponse 兼容的格式
	return models.KeyGenResponse{
		Success:      true,
		UserName:     req.UserName,
		Address:      address,
		EncryptedKey: encryptedKeyBase64,
	}, nil
}

// PerformSignMessage 执行消息签名
func (ws *WailsServices) PerformSignMessage(req models.SignRequest) (interface{}, error) {
	if !ws.initialized {
		if err := ws.Init(); err != nil {
			return nil, err
		}
	}

	// 验证地址格式并标准化（仿照原controller逻辑）
	address := req.Address
	if !strings.HasPrefix(address, "0x") {
		address = "0x" + address
	}

	// 解码Base64加密密钥（仿照原controller逻辑）
	encryptedKey, err := base64.StdEncoding.DecodeString(req.EncryptedKey)
	if err != nil {
		clog.Error("加密密钥解码失败", clog.String("error", err.Error()), clog.String("username", req.UserName))
		return nil, fmt.Errorf("加密密钥格式错误: %v", err)
	}

	// 解码签名（Base64格式）（仿照原controller逻辑）
	signature, err := base64.StdEncoding.DecodeString(req.Signature)
	if err != nil {
		clog.Error("签名解码失败", clog.String("error", err.Error()), clog.String("username", req.UserName))
		return nil, fmt.Errorf("签名格式错误: %v", err)
	}

	ctx := context.Background()
	// 使用解码后的数据调用服务，使用 models 中定义的字段
	result, err := ws.mpcService.SignMessage(
		ctx,
		req.Parties,
		req.Data,     // 使用 models 中的 Data 字段
		req.Filename, // 使用 models 中的 Filename 字段
		req.UserName,
		address,      // 使用标准化的地址
		encryptedKey, // 使用解码后的[]byte
		signature,    // 使用解码后的[]byte
	)
	if err != nil {
		return nil, err
	}

	// 确保以太坊签名格式（0x前缀）（仿照原controller逻辑）
	if !strings.HasPrefix(result, "0x") {
		result = "0x" + result
	}

	// 返回与原有 models.SignResponse 兼容的格式
	return models.SignResponse{
		Success:   true,
		Signature: result,
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
func (ws *WailsServices) PerformDeleteMessage(req models.DeleteRequest) error {
	if !ws.initialized {
		if err := ws.Init(); err != nil {
			return err
		}
	}

	// 验证地址格式并标准化（仿照原controller逻辑）
	address := req.Address
	if !strings.HasPrefix(address, "0x") {
		address = "0x" + address
	}

	// 解码签名（Base64格式）（仿照原controller逻辑）
	signature, err := base64.StdEncoding.DecodeString(req.Signature)
	if err != nil {
		clog.Error("签名解码失败", clog.String("error", err.Error()), clog.String("username", req.UserName))
		return fmt.Errorf("签名格式错误: %v", err)
	}

	// 调用 securityService 来执行删除，使用解码后的数据
	return ws.securityService.DeleteData(req.UserName, address, signature)
}
