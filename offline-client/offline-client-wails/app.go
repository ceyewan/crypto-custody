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
	// 在应用启动时初始化所有服务
	if err := a.wailsServices.Init(); err != nil {
		clog.Fatal("初始化Wails服务失败", clog.String("error", err.Error()))
	}
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
