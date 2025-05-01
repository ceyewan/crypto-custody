package manager

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"offline-server/clog"
)

// ProcessState 表示进程的状态
type ProcessState int32

const (
	// ProcessStopped 进程已停止
	ProcessStopped ProcessState = iota
	// ProcessRunning 进程正在运行
	ProcessRunning
	// ProcessRestarting 进程正在重启
	ProcessRestarting
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
	// GracefulTimeout 是优雅停止的超时时间
	GracefulTimeout time.Duration
	// Environment 进程的环境变量
	Environment []string
}

// DefaultConfig 返回默认的 Manager 配置
func DefaultConfig() Config {
	return Config{
		BinaryPath:      "./bin/gg20_sm_manager",
		LogDir:          "logs",
		RestartDelay:    3 * time.Second,
		AutoRestart:     true,
		GracefulTimeout: 5 * time.Second,
		Environment:     os.Environ(),
	}
}

// Process 表示一个受监控的 Manager 进程
type Process struct {
	config     Config
	cmd        *exec.Cmd
	logFile    *os.File
	done       chan struct{}
	state      int32 // 使用atomic操作，表示ProcessState
	lock       sync.Mutex
	logger     *clog.Logger
	cancelFunc context.CancelFunc // 用于取消监控goroutine
	startTime  time.Time
}

// New 创建一个新的 Manager 进程监控器
func New(config Config) *Process {
	// 使用默认值填充未指定的配置
	if config.BinaryPath == "" {
		config.BinaryPath = DefaultConfig().BinaryPath
	}
	if config.LogDir == "" {
		config.LogDir = DefaultConfig().LogDir
	}
	if config.RestartDelay <= 0 {
		config.RestartDelay = DefaultConfig().RestartDelay
	}
	if config.GracefulTimeout <= 0 {
		config.GracefulTimeout = DefaultConfig().GracefulTimeout
	}
	if len(config.Environment) == 0 {
		config.Environment = DefaultConfig().Environment
	}

	// 初始化日志
	logConfig := clog.DefaultConfig()
	logConfig.Filename = filepath.Join(config.LogDir, "manager_monitor.log")
	logConfig.ConsoleOutput = false
	logConfig.Level = clog.InfoLevel

	logger, err := clog.NewLogger(logConfig)
	if err != nil {
		fmt.Printf("初始化日志失败: %v, 使用标准输出\n", err)
	}

	return &Process{
		config: config,
		done:   make(chan struct{}),
		logger: logger,
		state:  int32(ProcessStopped),
	}
}

// Start 启动 Manager 进程并开始监控
func (mp *Process) Start() error {
	mp.lock.Lock()
	defer mp.lock.Unlock()

	// 检查进程是否已在运行
	if mp.GetState() == ProcessRunning {
		mp.logger.Info("Manager进程已在运行中")
		return nil
	}

	// 确保日志目录存在
	if err := os.MkdirAll(mp.config.LogDir, 0755); err != nil {
		mp.logger.Errorf("创建日志目录失败: %v", err)
		return fmt.Errorf("创建日志目录失败: %v", err)
	}

	mp.logger.Infof("开始启动 Manager 进程，二进制文件: %s", mp.config.BinaryPath)

	// 创建唯一的日志文件名
	logFileName := filepath.Join(mp.config.LogDir, fmt.Sprintf("manager-%s.log",
		time.Now().Format("20060102-150405")))

	logFile, err := os.Create(logFileName)
	if err != nil {
		mp.logger.Errorf("创建日志文件失败: %v", err)
		return fmt.Errorf("创建日志文件失败: %v", err)
	}

	// 关闭之前的日志文件（如果有）
	if mp.logFile != nil {
		mp.logFile.Close()
	}
	mp.logFile = logFile

	// 启动进程
	if err := mp.startProcess(); err != nil {
		mp.logFile.Close()
		mp.logFile = nil
		return err
	}

	// 创建符号链接指向最新的日志文件
	latestLogLink := filepath.Join(mp.config.LogDir, "manager-latest.log")
	os.Remove(latestLogLink) // 忽略错误，可能不存在
	if err := os.Symlink(logFileName, latestLogLink); err != nil {
		mp.logger.Warnf("创建日志符号链接失败: %v", err)
		// 继续执行，这不是致命错误
	}

	// 监控进程状态
	ctx, cancel := context.WithCancel(context.Background())
	mp.cancelFunc = cancel
	go mp.monitor(ctx)
	mp.logger.Info("Manager进程监控已启动")

	return nil
}

// startProcess 启动 Manager 进程
func (mp *Process) startProcess() error {
	cmd := exec.Command(mp.config.BinaryPath)
	cmd.Env = mp.config.Environment

	// 重定向标准输出和标准错误到日志文件
	cmd.Stdout = io.MultiWriter(mp.logFile, os.Stdout)
	cmd.Stderr = io.MultiWriter(mp.logFile, os.Stderr)

	// 启动进程
	if err := cmd.Start(); err != nil {
		mp.logger.Errorf("启动Manager失败: %v", err)
		return fmt.Errorf("启动Manager失败: %v", err)
	}

	mp.cmd = cmd
	mp.startTime = time.Now()
	atomic.StoreInt32(&mp.state, int32(ProcessRunning))
	mp.logger.Infof("Manager进程已启动，PID: %d", cmd.Process.Pid)

	return nil
}

// monitor 监控 Manager 进程，必要时重启
func (mp *Process) monitor(ctx context.Context) {
	mp.logger.Debug("开始监控 Manager 进程")

	for {
		// 等待进程退出或收到取消信号
		waitDone := make(chan error, 1)
		go func() {
			waitDone <- mp.cmd.Wait()
		}()

		select {
		case err := <-waitDone:
			// 进程退出
			exitTime := time.Now().Format("2006-01-02 15:04:05")

			mp.lock.Lock()
			if mp.GetState() == ProcessStopped {
				mp.logger.Info("进程已手动停止，退出监控")
				mp.lock.Unlock()
				return
			}

			if err != nil {
				mp.logger.Warnf("Manager进程异常退出: %v", err)
				fmt.Fprintf(mp.logFile, "[%s] Manager进程异常退出: %v\n", exitTime, err)
			} else {
				mp.logger.Info("Manager进程正常退出")
				fmt.Fprintf(mp.logFile, "[%s] Manager进程正常退出\n", exitTime)
			}

			// 更新状态
			atomic.StoreInt32(&mp.state, int32(ProcessStopped))

			// 如果需要自动重启
			if mp.config.AutoRestart {
				atomic.StoreInt32(&mp.state, int32(ProcessRestarting))
				restartMsg := fmt.Sprintf("%v 秒后尝试重启 Manager...", mp.config.RestartDelay.Seconds())
				mp.logger.Info(restartMsg)
				fmt.Fprintf(mp.logFile, "[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), restartMsg)
				mp.lock.Unlock()

				select {
				case <-time.After(mp.config.RestartDelay):
					mp.lock.Lock()
					if mp.GetState() == ProcessStopped {
						mp.logger.Info("监控已终止，取消重启")
						mp.lock.Unlock()
						return
					}

					mp.logger.Info("正在重启 Manager 进程")
					if err := mp.startProcess(); err != nil {
						errMsg := fmt.Sprintf("重启Manager失败: %v", err)
						mp.logger.Error(errMsg)
						fmt.Fprintf(mp.logFile, "[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), errMsg)
						atomic.StoreInt32(&mp.state, int32(ProcessStopped))
						mp.lock.Unlock()
						return
					}
					mp.lock.Unlock()

				case <-ctx.Done():
					mp.logger.Debug("接收到终止信号，取消重启")
					atomic.StoreInt32(&mp.state, int32(ProcessStopped))
					return
				}
			} else {
				mp.logger.Debug("自动重启已禁用，不再重启进程")
				mp.lock.Unlock()
				return
			}

		case <-ctx.Done():
			mp.logger.Debug("接收到监控终止信号")
			return
		}
	}
}

// Stop 停止 Manager 进程
func (mp *Process) Stop() error {
	mp.lock.Lock()
	defer mp.lock.Unlock()

	// 检查进程是否已经停止
	if mp.GetState() == ProcessStopped {
		mp.logger.Debug("Manager进程已经处于停止状态")
		return nil
	}

	// 标记状态为已停止
	atomic.StoreInt32(&mp.state, int32(ProcessStopped))

	// 取消监控
	if mp.cancelFunc != nil {
		mp.cancelFunc()
		mp.cancelFunc = nil
	}

	// 如果进程正在运行，尝试优雅终止
	if mp.cmd != nil && mp.cmd.Process != nil {
		mp.logger.Infof("正在终止Manager进程 (PID: %d)...", mp.cmd.Process.Pid)

		// 发送终止信号
		if err := mp.cmd.Process.Signal(os.Interrupt); err != nil {
			mp.logger.Warnf("发送中断信号失败: %v，尝试强制终止", err)
			return mp.cmd.Process.Kill()
		}

		// 给进程一点时间优雅退出
		done := make(chan error, 1)
		go func() {
			done <- mp.cmd.Wait()
		}()

		// 等待进程退出或超时
		select {
		case <-done:
			mp.logger.Info("Manager进程已正常终止")
		case <-time.After(mp.config.GracefulTimeout):
			mp.logger.Warn("Manager进程未能在时限内退出，强制终止")
			return mp.cmd.Process.Kill()
		}
	}

	// 关闭日志文件
	if mp.logFile != nil {
		mp.logger.Debug("关闭进程日志文件")
		err := mp.logFile.Close()
		mp.logFile = nil
		if err != nil {
			return fmt.Errorf("关闭日志文件失败: %v", err)
		}
	}

	// 记录日志
	if mp.logger != nil {
		mp.logger.Info("Manager 进程监控已停止")
		mp.logger.Sync()
	}

	return nil
}

// Restart 重启 Manager 进程
func (mp *Process) Restart() error {
	mp.logger.Info("正在重启 Manager 进程")

	// 先停止进程
	if err := mp.Stop(); err != nil {
		mp.logger.Errorf("停止进程失败: %v", err)
		return fmt.Errorf("重启失败，无法停止当前进程: %v", err)
	}

	// 等待一段时间再启动
	time.Sleep(mp.config.RestartDelay)

	// 启动新进程
	return mp.Start()
}

// IsRunning 检查进程是否在运行
func (mp *Process) IsRunning() bool {
	return mp.GetState() == ProcessRunning
}

// GetState 获取当前进程状态
func (mp *Process) GetState() ProcessState {
	return ProcessState(atomic.LoadInt32(&mp.state))
}

// GetUptime 获取进程运行时间
func (mp *Process) GetUptime() time.Duration {
	if !mp.IsRunning() || mp.startTime.IsZero() {
		return 0
	}
	return time.Since(mp.startTime)
}

// GetPID 获取进程ID
func (mp *Process) GetPID() int {
	mp.lock.Lock()
	defer mp.lock.Unlock()

	if mp.cmd != nil && mp.cmd.Process != nil {
		return mp.cmd.Process.Pid
	}
	return -1
}
