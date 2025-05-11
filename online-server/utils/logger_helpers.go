package utils

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/ceyewan/clog"
	"go.uber.org/zap"
)

// 增加 clog 没有的类型处理函数
var (
	Uint = zap.Uint // 创建无符号整数类型的日志字段
)

// GormLogWriter 实现了 io.Writer 和 logger.Writer 接口，用于将 GORM 日志重定向到 clog
type GormLogWriter struct {
	logger *clog.Logger
}

// Write 实现 io.Writer 接口
func (w *GormLogWriter) Write(p []byte) (n int, err error) {
	w.logger.Info(string(p))
	return len(p), nil
}

// Printf 实现 logger.Writer 接口
func (w *GormLogWriter) Printf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	w.logger.Info(msg)
}

// NewGormLogWriter 创建一个新的 GORM 日志写入器
func NewGormLogWriter() *GormLogWriter {
	return &GormLogWriter{
		logger: clog.Module("gorm"),
	}
}

// FormatDuration 格式化时间间隔为人类可读格式
func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	} else if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	} else {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
}

// GetLogWriter 返回适合当前环境的日志写入器
// 在开发环境返回标准输出，在生产环境返回 clog 写入器
func GetLogWriter() io.Writer {
	if os.Getenv("ENV") == "production" {
		return NewGormLogWriter()
	}
	return os.Stdout
}

// ErrString 从错误创建错误字符串，如果错误为 nil 则返回空字符串
func ErrString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// ShortenString 截断过长的字符串
func ShortenString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
