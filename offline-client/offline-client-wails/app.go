package main

import (
	"context"
	"fmt"
	"log"

	"offline-client-wails/clog"
	"offline-client-wails/config"
)

// App struct - Wails 应用程序结构体
type App struct {
	ctx      context.Context
	services *WailsServices
}

// NewApp 创建新的 App 应用程序结构体
func NewApp() *App {
	return &App{
		services: GetWailsServices(),
	}
}

// startup 当应用启动时调用，保存上下文以便调用运行时方法
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// 初始化配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("加载配置失败: %v", err)
		return
	}

	// 初始化日志系统
	if err = clog.Init(clog.DefaultConfig()); err != nil {
		log.Printf("初始化日志系统失败: %v", err)
		return
	}
	clog.SetDefaultLevel(clog.DebugLevel)

	clog.Info("Wails应用启动成功")
	clog.Info("配置加载成功",
		clog.String("port", cfg.Port),
		clog.Bool("debug", cfg.Debug),
		clog.String("log_file", cfg.LogFile),
		clog.String("log_dir", cfg.LogDir),
	)

	// 初始化服务
	if err := a.services.Init(); err != nil {
		log.Printf("初始化服务失败: %v", err)
	}
}

// Greet 为给定的名称返回问候语
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

// KeyGeneration 密钥生成 - 暴露给前端的方法
func (a *App) KeyGeneration() map[string]interface{} {
	clog.Info("前端调用密钥生成")

	result, err := a.services.PerformKeyGeneration()
	if err != nil {
		clog.Error("密钥生成失败", clog.String("error", err.Error()))
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
	}

	return map[string]interface{}{
		"success": true,
		"data":    result,
	}
}

// SignMessage 消息签名 - 暴露给前端的方法
func (a *App) SignMessage(message string) map[string]interface{} {
	clog.Info("前端调用消息签名", clog.String("message", message))

	result, err := a.services.PerformSignMessage(message)
	if err != nil {
		clog.Error("消息签名失败", clog.String("error", err.Error()))
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
	}

	return map[string]interface{}{
		"success": true,
		"data":    result,
	}
}

// GetCPLC 获取CPLC信息 - 暴露给前端的方法
func (a *App) GetCPLC() map[string]interface{} {
	clog.Info("前端调用获取CPLC信息")

	result, err := a.services.GetCPLCInfo()
	if err != nil {
		clog.Error("获取CPLC信息失败", clog.String("error", err.Error()))
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
	}

	return map[string]interface{}{
		"success": true,
		"data":    result,
	}
}

// DeleteMessage 删除消息 - 暴露给前端的方法
func (a *App) DeleteMessage() map[string]interface{} {
	clog.Info("前端调用删除消息")

	err := a.services.PerformDeleteMessage()
	if err != nil {
		clog.Error("删除消息失败", clog.String("error", err.Error()))
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
	}

	return map[string]interface{}{
		"success": true,
		"message": "删除成功",
	}
}
