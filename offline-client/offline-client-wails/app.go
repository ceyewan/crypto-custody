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

// SaveFile prompts the operator for a path and writes text content to disk.
func (a *App) SaveFile(defaultFileName string, content string) (string, error) {
	if a.ctx == nil {
		return "", errors.New("应用尚未初始化")
	}

	defaultFileName = strings.TrimSpace(defaultFileName)
	if defaultFileName == "" {
		defaultFileName = "backup.dat"
	}

	savePath, err := wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:                "保存文件",
		DefaultDirectory:     downloadsDir(),
		DefaultFilename:      defaultFileName,
		CanCreateDirectories: true,
		Filters: []wailsruntime.FileFilter{
			{
				DisplayName: "所有文件 (*.*)",
				Pattern:     "*.*",
			},
		},
	})
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(savePath) == "" {
		return "", nil
	}

	if err := os.WriteFile(savePath, []byte(content), 0600); err != nil {
		return "", err
	}
	return savePath, nil
}

// OpenFile prompts the operator to choose a file and returns its name and text content.
func (a *App) OpenFile() (map[string]string, error) {
	if a.ctx == nil {
		return nil, errors.New("应用尚未初始化")
	}

	path, err := wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title:            "选择冷备份文件",
		DefaultDirectory: downloadsDir(),
		Filters: []wailsruntime.FileFilter{
			{
				DisplayName: "冷备份文件 (*.enc)",
				Pattern:     "*.enc",
			},
			{
				DisplayName: "所有文件 (*.*)",
				Pattern:     "*.*",
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(path) == "" {
		return map[string]string{}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"path":     path,
		"fileName": filepath.Base(path),
		"content":  string(data),
	}, nil
}

func downloadsDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, "Downloads")
}
