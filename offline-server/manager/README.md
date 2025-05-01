# Manager 进程管理器

## 简介
`manager` 包提供了一个进程管理器，用于启动、停止、重启和监控后台进程。它支持自动重启、日志记录和优雅停止等功能。

## 功能
- **启动**：启动指定的后台进程。
- **停止**：优雅停止正在运行的进程。
- **重启**：停止后重新启动进程。
- **监控**：监控进程状态，支持异常退出后的自动重启。

## 快速开始

### 1. 初始化配置
创建一个 `Config` 对象并设置必要的参数：

```go
config := manager.Config{
    BinaryPath:      "./bin/gg20_sm_manager", // 可执行文件路径
    LogDir:          "logs",                 // 日志目录
    RestartDelay:    3 * time.Second,         // 重启延迟时间
    AutoRestart:     true,                    // 是否自动重启
    GracefulTimeout: 5 * time.Second,         // 优雅停止超时时间
    Environment:     os.Environ(),            // 环境变量
}
```

### 2. 创建进程管理器
使用 `New` 方法创建一个 `Process` 实例：

```go
process := manager.New(config)
```

### 3. 启动进程
调用 `Start` 方法启动进程：

```go
if err := process.Start(); err != nil {
    fmt.Printf("启动失败: %v\n", err)
}
```

### 4. 停止进程
调用 `Stop` 方法优雅停止进程：

```go
if err := process.Stop(); err != nil {
    fmt.Printf("停止失败: %v\n", err)
}
```

### 5. 重启进程
调用 `Restart` 方法重启进程：

```go
if err := process.Restart(); err != nil {
    fmt.Printf("重启失败: %v\n", err)
}
```

### 6. 检查状态
- 检查进程是否正在运行：

```go
if process.IsRunning() {
    fmt.Println("进程正在运行")
}
```

- 获取进程运行时间：

```go
uptime := process.GetUptime()
fmt.Printf("进程已运行时间: %v\n", uptime)
```

- 获取进程 ID：

```go
pid := process.GetPID()
fmt.Printf("进程 PID: %d\n", pid)
```

## 日志
日志文件默认存储在 `logs` 目录下，最新的日志文件会通过符号链接 `manager-latest.log` 指向。

## 注意事项
- 确保 `BinaryPath` 指向的可执行文件存在并具有执行权限。
- 如果启用了自动重启功能，请确保 `RestartDelay` 设置合理，避免频繁重启。

## 许可证
MIT License