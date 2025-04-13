package ws

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// ManagerConfig 配置 Manager 进程的行为
type ManagerConfig struct {
	// Manager 可执行文件的路径
	BinaryPath string
	// 日志文件的目录
	LogDir string
	// 重启前等待的时间（秒）
	RestartDelay time.Duration
	// 进程异常退出后是否自动重启
	AutoRestart bool
}

// DefaultManagerConfig 返回默认的 Manager 配置
func DefaultManagerConfig() ManagerConfig {
	return ManagerConfig{
		BinaryPath:   "./gg20_sm_manager",
		LogDir:       "logs",
		RestartDelay: 3 * time.Second,
		AutoRestart:  true,
	}
}

// ManagerProcess 表示一个受监控的 Manager 进程
type ManagerProcess struct {
	config     ManagerConfig
	cmd        *exec.Cmd
	logFile    *os.File
	done       chan struct{}
	terminated bool
	lock       sync.Mutex
}

// NewManagerProcess 创建一个新的 Manager 进程监控器
func NewManagerProcess(config ManagerConfig) *ManagerProcess {
	if config.BinaryPath == "" {
		config.BinaryPath = "./gg20_sm_manager"
	}
	if config.LogDir == "" {
		config.LogDir = "logs"
	}
	if config.RestartDelay <= 0 {
		config.RestartDelay = 3 * time.Second
	}

	return &ManagerProcess{
		config: config,
		done:   make(chan struct{}),
	}
}

// Start 启动 Manager 进程并开始监控
func (mp *ManagerProcess) Start() error {
	mp.lock.Lock()
	defer mp.lock.Unlock()

	// 确保日志目录存在
	if err := os.MkdirAll(mp.config.LogDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 创建唯一的日志文件名
	logFileName := filepath.Join(mp.config.LogDir, fmt.Sprintf("manager-%s.log",
		time.Now().Format("20060102-150405")))

	logFile, err := os.Create(logFileName)
	if err != nil {
		return fmt.Errorf("创建日志文件失败: %v", err)
	}
	mp.logFile = logFile

	// 启动进程
	if err := mp.startProcess(); err != nil {
		mp.logFile.Close()
		return err
	}

	// 创建符号链接指向最新的日志文件
	latestLogLink := filepath.Join(mp.config.LogDir, "manager-latest.log")
	os.Remove(latestLogLink) // 忽略错误，可能不存在
	if err := os.Symlink(logFileName, latestLogLink); err != nil {
		log.Printf("创建日志符号链接失败: %v", err)
		// 继续执行，这不是致命错误
	}

	// 监控进程状态
	go mp.monitor()

	return nil
}

// startProcess 启动 Manager 进程
func (mp *ManagerProcess) startProcess() error {
	cmd := exec.Command(mp.config.BinaryPath)

	// 重定向标准输出和标准错误到日志文件
	cmd.Stdout = io.MultiWriter(mp.logFile, os.Stdout)
	cmd.Stderr = io.MultiWriter(mp.logFile, os.Stderr)

	// 启动进程
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动Manager失败: %v", err)
	}

	mp.cmd = cmd
	log.Printf("Manager进程已启动，PID: %d", cmd.Process.Pid)

	return nil
}

// monitor 监控 Manager 进程，必要时重启
func (mp *ManagerProcess) monitor() {
	for {
		// 等待进程退出
		err := mp.cmd.Wait()

		mp.lock.Lock()
		if mp.terminated {
			mp.lock.Unlock()
			return
		}

		// 进程退出
		exitTime := time.Now().Format("2006-01-02 15:04:05")
		if err != nil {
			fmt.Fprintf(mp.logFile, "[%s] Manager进程异常退出: %v\n", exitTime, err)
			log.Printf("Manager进程异常退出: %v", err)
		} else {
			fmt.Fprintf(mp.logFile, "[%s] Manager进程正常退出\n", exitTime)
			log.Printf("Manager进程正常退出")
		}

		// 如果配置了自动重启，则重启进程
		if mp.config.AutoRestart {
			fmt.Fprintf(mp.logFile, "[%s] %v 秒后尝试重启 Manager...\n",
				time.Now().Format("2006-01-02 15:04:05"), mp.config.RestartDelay.Seconds())
			log.Printf("%.1f 秒后尝试重启 Manager...", mp.config.RestartDelay.Seconds())

			mp.lock.Unlock()

			// 等待指定时间后重启
			select {
			case <-time.After(mp.config.RestartDelay):
				mp.lock.Lock()
				if mp.terminated {
					mp.lock.Unlock()
					return
				}

				// 重启进程
				if err := mp.startProcess(); err != nil {
					fmt.Fprintf(mp.logFile, "[%s] 重启Manager失败: %v\n",
						time.Now().Format("2006-01-02 15:04:05"), err)
					log.Printf("重启Manager失败: %v", err)
					mp.lock.Unlock()
					return
				}
				mp.lock.Unlock()

			case <-mp.done:
				return
			}
		} else {
			mp.lock.Unlock()
			return
		}
	}
}

// Stop 停止 Manager 进程
func (mp *ManagerProcess) Stop() {
	mp.lock.Lock()
	defer mp.lock.Unlock()

	if mp.terminated {
		return
	}

	// 标记已终止
	mp.terminated = true
	close(mp.done)

	// 如果进程正在运行，尝试优雅终止
	if mp.cmd != nil && mp.cmd.Process != nil {
		log.Printf("正在终止Manager进程 (PID: %d)...", mp.cmd.Process.Pid)

		// 发送终止信号
		if err := mp.cmd.Process.Signal(os.Interrupt); err != nil {
			log.Printf("发送中断信号失败: %v，尝试强制终止", err)
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
			log.Printf("Manager进程已终止")
		case <-time.After(5 * time.Second):
			log.Printf("Manager进程未能及时退出，强制终止")
			mp.cmd.Process.Kill()
		}
	}

	// 关闭日志文件
	if mp.logFile != nil {
		mp.logFile.Close()
		mp.logFile = nil
	}
}

// RunManager 启动Manager进程并实现自动重启
// 为兼容现有代码，保留原函数名称但实现更健壮的进程管理
func RunManager() (*exec.Cmd, error) {
	// 使用默认配置
	config := DefaultManagerConfig()

	// 创建监控器并启动Manager进程
	manager := NewManagerProcess(config)
	if err := manager.Start(); err != nil {
		return nil, err
	}

	// 返回兼容的命令实例
	// 注意：外部代码仍可使用返回的exec.Cmd，但实际的进程管理由ManagerProcess负责
	return manager.cmd, nil
}
