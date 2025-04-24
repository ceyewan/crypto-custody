package utils

import (
	"fmt"
	"path/filepath"
	"sync"

	"web-se/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logger *zap.Logger
	once   sync.Once
)

// InitLogger 初始化日志系统
func InitLogger(cfg *config.Config) (*zap.Logger, error) {
	var err error

	once.Do(func() {
		// 确保日志目录存在
		if err = EnsureDir(cfg.LogDir); err != nil {
			err = fmt.Errorf("创建日志目录失败: %v", err)
			return
		}

		// 创建日志轮转配置
		logPath := filepath.Join(cfg.LogDir, cfg.LogFile)
		w := zapcore.AddSync(&lumberjack.Logger{
			Filename:   logPath,
			MaxSize:    cfg.LogMaxSize,    // MB
			MaxBackups: cfg.LogMaxBackups, // 保留的旧日志文件数量
			MaxAge:     cfg.LogMaxAge,     // 天
			Compress:   cfg.LogCompress,   // 是否压缩
		})

		// 配置编码器
		encoderConfig := zap.NewProductionEncoderConfig()
		encoderConfig.TimeKey = "time"
		encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		encoderConfig.EncodeDuration = zapcore.StringDurationEncoder
		encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

		// 创建核心
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			w,
			getLogLevel(cfg.Debug),
		)

		// 创建日志记录器
		logger = zap.New(
			core,
			zap.AddCaller(),
			zap.AddCallerSkip(1),
			zap.AddStacktrace(zapcore.ErrorLevel),
		)
	})

	return logger, err
}

// GetLogger 获取日志实例
func GetLogger() *zap.Logger {
	if logger == nil {
		// 如果未初始化，返回一个基本的控制台日志记录器
		logger, _ = zap.NewProduction()
	}
	return logger
}

// getLogLevel 根据调试标志返回日志级别
func getLogLevel(debug bool) zapcore.Level {
	if debug {
		return zapcore.DebugLevel
	}
	return zapcore.InfoLevel
}

// LogDebug 记录调试日志
func LogDebug(msg string, fields ...zap.Field) {
	GetLogger().Debug(msg, fields...)
}

// LogInfo 记录信息日志
func LogInfo(msg string, fields ...zap.Field) {
	GetLogger().Info(msg, fields...)
}

// LogWarn 记录警告日志
func LogWarn(msg string, fields ...zap.Field) {
	GetLogger().Warn(msg, fields...)
}

// LogError 记录错误日志
func LogError(msg string, fields ...zap.Field) {
	GetLogger().Error(msg, fields...)
}

// LogFatal 记录致命错误日志
func LogFatal(msg string, fields ...zap.Field) {
	GetLogger().Fatal(msg, fields...)
}

// Field 快捷函数 - 字符串字段
func String(key, val string) zap.Field {
	return zap.String(key, val)
}

// Int 快捷函数 - 整数字段
func Int(key string, val int) zap.Field {
	return zap.Int(key, val)
}

// Error 快捷函数 - 错误字段
func Error(err error) zap.Field {
	return zap.Error(err)
}

// Bool 快捷函数 - 布尔字段
func Bool(key string, val bool) zap.Field {
	return zap.Bool(key, val)
}

// Binary 快捷函数 - 二进制数据字段
func Binary(key string, val []byte) zap.Field {
	return zap.Binary(key, val)
}

// Any 快捷函数 - 任意类型字段
func Any(key string, val interface{}) zap.Field {
	return zap.Any(key, val)
}
