package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
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
	initMu          sync.Mutex
	initialized     bool
	cardReaderName  string
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
	ws.initMu.Lock()
	defer ws.initMu.Unlock()

	if ws.initialized {
		return nil
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		clog.Error("加载配置失败", clog.String("error", err.Error()))
		return err
	}
	if ws.cardReaderName != "" {
		cfg.CardReaderName = ws.cardReaderName
	}
	if err := applyDesktopRuntimePaths(cfg); err != nil {
		clog.Error("准备桌面端运行目录失败", clog.String("error", err.Error()))
		return err
	}

	securityService, err := services.NewSecurityService(cfg)
	if err != nil {
		clog.Error("创建安全芯片服务失败", clog.String("error", err.Error()))
		return err
	}

	ws.cfg = cfg
	ws.securityService = securityService
	ws.mpcService = services.NewMPCService(cfg, securityService)
	ws.initialized = true
	clog.Info("WailsServices初始化成功",
		clog.String("card_reader_name", cfg.CardReaderName),
		clog.String("temp_dir", cfg.TempDir),
		clog.String("log_dir", cfg.LogDir))
	return nil
}

func applyDesktopRuntimePaths(cfg *config.Config) error {
	baseDir, err := os.UserCacheDir()
	if err != nil || strings.TrimSpace(baseDir) == "" {
		baseDir = os.TempDir()
	}

	appDir := filepath.Join(baseDir, "crypto-custody", "offline-client")
	if strings.TrimSpace(cfg.TempDir) == "" || !filepath.IsAbs(cfg.TempDir) {
		cfg.TempDir = filepath.Join(appDir, "mpc-temp", fmt.Sprintf("pid-%d", os.Getpid()))
	}
	if strings.TrimSpace(cfg.LogDir) == "" || !filepath.IsAbs(cfg.LogDir) {
		cfg.LogDir = filepath.Join(appDir, "logs")
	}

	for _, dir := range []string{cfg.TempDir, cfg.LogDir} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("创建运行目录失败 %s: %w", dir, err)
		}
	}
	return nil
}

// SetCardReaderName 设置读卡器名称。已初始化后也会立即影响后续 SE 操作。
func (ws *WailsServices) SetCardReaderName(name string) error {
	ws.initMu.Lock()
	defer ws.initMu.Unlock()

	name = strings.TrimSpace(name)
	ws.cardReaderName = name
	if ws.cfg != nil {
		ws.cfg.CardReaderName = name
	}
	clog.Info("读卡器名称已更新", clog.String("card_reader_name", name))
	return nil
}

// GetCardReaderName 返回当前读卡器名称。
func (ws *WailsServices) GetCardReaderName() string {
	ws.initMu.Lock()
	defer ws.initMu.Unlock()

	if ws.cardReaderName != "" {
		return ws.cardReaderName
	}
	if ws.cfg != nil {
		return ws.cfg.CardReaderName
	}
	cfg, err := config.LoadConfig()
	if err != nil {
		clog.Warn("读取读卡器名称失败", clog.String("error", err.Error()))
		return ""
	}
	return cfg.CardReaderName
}

// PerformKeyGeneration 执行密钥生成
func (ws *WailsServices) PerformKeyGeneration(req models.KeyGenRequest) (interface{}, error) {
	if !ws.initialized {
		if err := ws.Init(); err != nil {
			return nil, err
		}
	}

	ctx := context.Background()
	address, publicKey, encryptedShard, err := ws.mpcService.KeyGeneration(ctx, req.ManagerAddr, req.Room, req.Threshold, req.Parties, req.PartyIndex, req.Filename, req.RecordID)
	if err != nil {
		return nil, err
	}

	// 确保地址格式正确（添加0x前缀如果没有）
	if !strings.HasPrefix(address, "0x") {
		address = "0x" + address
	}

	// 转换加密分片为Base64字符串（仿照原controller逻辑）
	encryptedShardBase64 := base64.StdEncoding.EncodeToString(encryptedShard)

	return models.KeyGenResponse{
		Success:        true,
		Address:        address,
		PublicKey:      publicKey,
		RecordID:       req.RecordID,
		EncryptedShard: encryptedShardBase64,
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

	// 解码Base64加密分片（仿照原controller逻辑）
	encryptedShard, err := base64.StdEncoding.DecodeString(req.EncryptedShard)
	if err != nil {
		clog.Error("加密分片解码失败", clog.String("error", err.Error()))
		return nil, fmt.Errorf("加密分片格式错误: %v", err)
	}

	// 解码签名（Base64格式）（仿照原controller逻辑）
	signature, err := base64.StdEncoding.DecodeString(req.Signature)
	if err != nil {
		clog.Error("签名解码失败", clog.String("error", err.Error()))
		return nil, fmt.Errorf("签名格式错误: %v", err)
	}

	ctx := context.Background()
	// 使用解码后的数据调用服务，使用 models 中定义的字段
	result, err := ws.mpcService.SignMessage(
		ctx,
		req.ManagerAddr,
		req.Room,
		req.SigningIndex,
		req.Parties,
		req.MessageHash,
		req.Filename,
		req.RecordID,
		address,
		encryptedShard,
		signature,
	)
	if err != nil {
		return nil, err
	}

	// 确保以太坊签名格式（0x前缀）（仿照原controller逻辑）
	if !strings.HasPrefix(result, "0x") {
		result = "0x" + result
	}

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
		"cplc_info": strings.ToUpper(hex.EncodeToString(cplcInfo)),
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
		clog.Error("签名解码失败", clog.String("error", err.Error()))
		return fmt.Errorf("签名格式错误: %v", err)
	}

	if err := ws.securityService.DeleteData(req.RecordID, address, signature); err != nil {
		return err
	}

	if _, err := ws.securityService.ReadData(req.RecordID, address, signature); err == nil {
		return fmt.Errorf("SE记录删除后仍可读取")
	}
	return nil
}
