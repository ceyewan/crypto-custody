package utils

import (
	"os"
	"path/filepath"
	"time"

	"github.com/ceyewan/clog"
)

// Exported module loggers
var (
	AccountLogger *clog.Logger
	UserLogger    *clog.Logger
	SystemLogger  *clog.Logger
)

// 初始化日志
func InitLogger() error {
	// 确保日志目录存在
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return err
	}

	// 创建日志配置
	config := clog.DefaultConfig()

	// 根据环境配置日志
	if os.Getenv("ENV") == "production" {
		// 生产环境: JSON格式，不输出到控制台，带文件轮转
		config.Level = clog.InfoLevel
		config.Format = clog.FormatJSON
		config.ConsoleOutput = false
		config.EnableColor = false
		config.Filename = filepath.Join(logsDir, "app.log")
		config.FileRotation = &clog.FileRotationConfig{
			MaxSize:    100,  // 单个文件最大 100MB
			MaxAge:     30,   // 保留 30 天
			MaxBackups: 10,   // 最多保留 10 个文件
			Compress:   true, // 压缩旧文件
		}
	} else {
		// 开发环境: 控制台友好格式，同时输出到文件，带颜色
		config.Level = clog.DebugLevel
		config.Format = clog.FormatConsole
		config.ConsoleOutput = true
		config.EnableColor = true
		config.EnableCaller = true
		config.UseTimeStampFilename = true
		config.Filename = filepath.Join(logsDir, "app.log")
	}

	// 初始化日志
	if err := clog.Init(config); err != nil {
		return err
	}

	// 创建业务子模块日志器
	createModuleLoggers()

	// 记录启动信息
	clog.Info("系统启动",
		clog.String("time", time.Now().Format(time.RFC3339)),
		clog.String("env", getEnv("ENV", "development")),
	)

	return nil
}

// 创建模块日志器
func createModuleLoggers() {
	// 账户模块日志
	AccountLogger = clog.Module("account", clog.Config{
		Level:         clog.InfoLevel,
		EnableCaller:  true,
		ConsoleOutput: true,
	})
	AccountLogger.Info("账户模块日志器初始化成功")

	// 用户模块日志
	UserLogger = clog.Module("user", clog.Config{
		Level:         clog.InfoLevel,
		EnableCaller:  true,
		ConsoleOutput: true,
	})
	UserLogger.Info("用户模块日志器初始化成功")

	// 系统模块日志
	SystemLogger = clog.Module("system", clog.Config{
		Level:         clog.InfoLevel,
		EnableCaller:  true,
		ConsoleOutput: true,
	})
	SystemLogger.Info("系统模块日志器初始化成功")
}

// 获取环境变量值，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// FlushLogs 刷新所有日志
func FlushLogs() {
	clog.SyncAll()
}
