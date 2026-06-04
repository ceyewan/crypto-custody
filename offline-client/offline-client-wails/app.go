package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"offline-client-wails/mpc_core/clog"
	"offline-client-wails/mpc_core/models"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx           context.Context
	wailsServices *WailsServices
}

// NewApp 创建App实例
func NewApp() *App {
	return &App{
		wailsServices: GetWailsServices(),
	}
}

// startup 是应用启动时调用的函数
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	clog.Info("Wails应用已启动，MPC/SE服务将在任务执行时初始化")
}

// 以下是需要绑定到前端的方法

// PerformKeyGeneration 绑定密钥生成方法
func (a *App) PerformKeyGeneration(req models.KeyGenRequest) (interface{}, error) {
	return a.wailsServices.PerformKeyGeneration(req)
}

// PerformSignMessage 绑定消息签名方法
func (a *App) PerformSignMessage(req models.SignRequest) (interface{}, error) {
	return a.wailsServices.PerformSignMessage(req)
}

// GetCPLCInfo 绑定获取CPLC信息的方法
func (a *App) GetCPLCInfo() (interface{}, error) {
	return a.wailsServices.GetCPLCInfo()
}

// PerformDeleteMessage 绑定删除消息的方法
func (a *App) PerformDeleteMessage(req models.DeleteRequest) error {
	return a.wailsServices.PerformDeleteMessage(req)
}

// SetCardReaderName 设置当前桌面端使用的读卡器名称。
func (a *App) SetCardReaderName(name string) error {
	return a.wailsServices.SetCardReaderName(name)
}

// GetCardReaderName 获取当前桌面端使用的读卡器名称。
func (a *App) GetCardReaderName() string {
	return a.wailsServices.GetCardReaderName()
}

// SaveJSONFile prompts the operator for a path and writes a JSON result package to disk.
func (a *App) SaveJSONFile(defaultFileName string, content string) (string, error) {
	if a.ctx == nil {
		return "", errors.New("应用尚未初始化")
	}

	defaultFileName = strings.TrimSpace(defaultFileName)
	if defaultFileName == "" {
		defaultFileName = "offline_result.json"
	}
	if !strings.HasSuffix(strings.ToLower(defaultFileName), ".json") {
		defaultFileName += ".json"
	}

	savePath, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:                "保存 JSON 结果包",
		DefaultDirectory:     downloadsDir(),
		DefaultFilename:      defaultFileName,
		CanCreateDirectories: true,
		Filters: []wailsruntime.FileFilter{
			{
				DisplayName: "JSON 文件 (*.json)",
				Pattern:     "*.json",
			},
		},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(savePath) == "" {
		return "", nil
	}

	if err := os.WriteFile(savePath, []byte(content), 0644); err != nil {
		return "", err
	}
	return savePath, nil
}

func downloadsDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, "Downloads")
}
