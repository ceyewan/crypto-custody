package manager

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"offline-server/clog"
)

// Config 配置 Manager 进程的行为
type Config struct {
	// BinaryPath 是 Manager 可执行文件的路径
	BinaryPath string
	// LogDir 是日志文件的目录
	LogDir string
	// RestartDelay 是重启前等待的时间
	RestartDelay time.Duration
	// AutoRestart 控制进程异常退出后是否自动重启
	AutoRestart bool
}

// DefaultConfig 返回默认的 Manager 配置
func DefaultConfig() Config {
	return Config{
		BinaryPath:   "./bin/gg20_sm_manager",
		LogDir:       "logs",
		RestartDelay: 3 * time.Second,
		AutoRestart:  true,
	}
}

// Process 表示一个受监控的 Manager 进程
type Process struct {
	config     Config
	cmd        *exec.Cmd
	logFile    *os.File
	done       chan struct{}
	terminated bool
	lock       sync.Mutex
	logger     *clog.Logger
}

// New 创建一个新的 Manager 进程监控器
// 使用指定的配置初始化进程监控，如果配置中有未指定的值，则使用默认值
func New(config Config) *Process {
	if config.BinaryPath == "" {
		config.BinaryPath = "./bin/gg20_sm_manager"
	}
	if config.LogDir == "" {
		config.LogDir = "logs"
	}
	if config.RestartDelay <= 0 {
		config.RestartDelay = 3 * time.Second
	}

	// 初始化日志
	logConfig := clog.DefaultConfig()
	logConfig.Filename = filepath.Join(config.LogDir, "manager_monitor.log")
	logConfig.ConsoleOutput = false
	logConfig.Level = clog.InfoLevel

	logger, err := clog.NewLogger(logConfig)
	if err != nil {
		// 如果日志初始化失败，至少打印错误信息
		fmt.Printf("初始化日志失败: %v, 使用标准输出\n", err)
	}

	return &Process{
		config: config,
		done:   make(chan struct{}),
		logger: logger,
	}
}

// Start 启动 Manager 进程并开始监控
// 创建日志目录，启动进程，并开始监控进程状态
func (mp *Process) Start() error {
	mp.lock.Lock()
	defer mp.lock.Unlock()

	// 确保日志目录存在
	if err := os.MkdirAll(mp.config.LogDir, 0755); err != nil {
		mp.logger.Errorf("创建日志目录失败: %v", err)
		return fmt.Errorf("创建日志目录失败: %v", err)
	}

	mp.logger.Infof("开始启动 Manager 进程，二进制文件: %s", mp.config.BinaryPath)
	mp.logger.Debugf("使用配置: %+v", mp.config)

	// 创建唯一的日志文件名
	logFileName := filepath.Join(mp.config.LogDir, fmt.Sprintf("manager-%s.log",
		time.Now().Format("20060102-150405")))

	logFile, err := os.Create(logFileName)
	if err != nil {
		mp.logger.Errorf("创建日志文件失败: %v", err)
		return fmt.Errorf("创建日志文件失败: %v", err)
	}
	mp.logFile = logFile
	mp.logger.Debugf("创建日志文件: %s", logFileName)

	// 启动进程
	if err := mp.startProcess(); err != nil {
		mp.logFile.Close()
		return err
	}

	// 创建符号链接指向最新的日志文件
	latestLogLink := filepath.Join(mp.config.LogDir, "manager-latest.log")
	os.Remove(latestLogLink) // 忽略错误，可能不存在
	if err := os.Symlink(logFileName, latestLogLink); err != nil {
		mp.logger.Warnf("创建日志符号链接失败: %v", err)
		// 继续执行，这不是致命错误
	} else {
		mp.logger.Debugf("创建符号链接: %s -> %s", latestLogLink, logFileName)
	}

	// 监控进程状态
	go mp.monitor()
	mp.logger.Info("Manager进程监控已启动")

	return nil
}

// startProcess 启动 Manager 进程
func (mp *Process) startProcess() error {
	cmd := exec.Command(mp.config.BinaryPath)
	mp.logger.Debugf("准备启动命令: %s", mp.config.BinaryPath)

	// 重定向标准输出和标准错误到日志文件
	cmd.Stdout = io.MultiWriter(mp.logFile, os.Stdout)
	cmd.Stderr = io.MultiWriter(mp.logFile, os.Stderr)

	// 启动进程
	if err := cmd.Start(); err != nil {
		mp.logger.Errorf("启动Manager失败: %v", err)
		return fmt.Errorf("启动Manager失败: %v", err)
	}

	mp.cmd = cmd
	mp.logger.Infof("Manager进程已启动，PID: %d", cmd.Process.Pid)

	return nil
}

// monitor 监控 Manager 进程，必要时重启
// 这是一个内部方法，在单独的goroutine中运行
func (mp *Process) monitor() {
	mp.logger.Debug("开始监控 Manager 进程")

	for {
		// 等待进程退出
		err := mp.cmd.Wait()

		mp.lock.Lock()
		if mp.terminated {
			mp.logger.Info("监控已终止，停止进程监控")
			mp.lock.Unlock()
			return
		}

		// 进程退出
		exitTime := time.Now().Format("2006-01-02 15:04:05")
		if err != nil {
			mp.logger.Warnf("Manager进程异常退出: %v", err)
			fmt.Fprintf(mp.logFile, "[%s] Manager进程异常退出: %v\n", exitTime, err)
		} else {
			mp.logger.Info("Manager进程正常退出")
			fmt.Fprintf(mp.logFile, "[%s] Manager进程正常退出\n", exitTime)
		}

		// 如果配置了自动重启，则重启进程
		if mp.config.AutoRestart {
			restartMsg := fmt.Sprintf("%v 秒后尝试重启 Manager...", mp.config.RestartDelay.Seconds())
			mp.logger.Info(restartMsg)
			fmt.Fprintf(mp.logFile, "[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), restartMsg)

			mp.lock.Unlock()

			// 等待指定时间后重启
			select {
			case <-time.After(mp.config.RestartDelay):
				mp.lock.Lock()
				if mp.terminated {
					mp.logger.Info("监控已终止，取消重启")
					mp.lock.Unlock()
					return
				}

				// 重启进程
				mp.logger.Info("正在重启 Manager 进程")
				if err := mp.startProcess(); err != nil {
					errMsg := fmt.Sprintf("重启Manager失败: %v", err)
					mp.logger.Error(errMsg)
					fmt.Fprintf(mp.logFile, "[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), errMsg)
					mp.lock.Unlock()
					return
				}
				mp.lock.Unlock()

			case <-mp.done:
				mp.logger.Debug("接收到终止信号，取消重启")
				return
			}
		} else {
			mp.logger.Debug("自动重启已禁用，不再重启进程")
			mp.lock.Unlock()
			return
		}
	}
}

// Stop 停止 Manager 进程
// 尝试优雅地终止正在运行的进程，并清理资源
func (mp *Process) Stop() {
	mp.lock.Lock()
	defer mp.lock.Unlock()

	if mp.terminated {
		mp.logger.Debug("Stop 被调用，但进程已终止")
		return
	}

	// 标记已终止
	mp.terminated = true
	close(mp.done)
	mp.logger.Info("正在停止 Manager 进程监控")

	// 如果进程正在运行，尝试优雅终止
	if mp.cmd != nil && mp.cmd.Process != nil {
		mp.logger.Infof("正在终止Manager进程 (PID: %d)...", mp.cmd.Process.Pid)

		// 发送终止信号
		if err := mp.cmd.Process.Signal(os.Interrupt); err != nil {
			mp.logger.Warnf("发送中断信号失败: %v，尝试强制终止", err)
			mp.cmd.Process.Kill()
		}

		// 给进程一点时间优雅退出
		done := make(chan error, 1)
		go func() {
			done <- mp.cmd.Wait()
		}()

		// 等待最多5秒
		select {
		case <-done:
			mp.logger.Info("Manager进程已正常终止")
		case <-time.After(5 * time.Second):
			mp.logger.Warn("Manager进程未能及时退出，强制终止")
			mp.cmd.Process.Kill()
		}
	}

	// 关闭日志文件
	if mp.logFile != nil {
		mp.logger.Debug("关闭进程日志文件")
		mp.logFile.Close()
		mp.logFile = nil
	}

	// 关闭监控日志
	if mp.logger != nil {
		mp.logger.Info("Manager 进程监控已停止")
		mp.logger.Sync()
	}
}

// GetCommand 返回当前运行的命令实例
// 这个方法用于兼容旧代码，提供对底层命令对象的访问
func (mp *Process) GetCommand() *exec.Cmd {
	mp.lock.Lock()
	defer mp.lock.Unlock()
	return mp.cmd
}
