package main

import (
	"context"

	"offline-client-wails/mpc_core/clog"
	"offline-client-wails/mpc_core/models"
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
