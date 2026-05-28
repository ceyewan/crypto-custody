package manager

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Session 描述一次 keygen/sign 会话独占的 gg20 manager。
type Session struct {
	SessionKey string
	ManagerURL string
	Room       string
	Port       int
}

// SessionRuntime 管理会话级 gg20 manager 生命周期。
type SessionRuntime interface {
	StartSession(sessionKey string) (Session, error)
	StopSession(sessionKey string) error
	StopAll() error
}

// SessionRuntimeConfig 是会话级 gg20 manager 的运行配置。
type SessionRuntimeConfig struct {
	BinaryPath      string
	BindAddress     string
	PublicHost      string
	LogDir          string
	PortStart       int
	PortEnd         int
	GracefulTimeout time.Duration
	Environment     []string
}

// DefaultSessionRuntimeConfig 返回可通过环境变量覆盖的默认配置。
func DefaultSessionRuntimeConfig() SessionRuntimeConfig {
	cfg := SessionRuntimeConfig{
		BinaryPath:      envString("OFFLINE_MANAGER_BIN", defaultManagerBinaryPath()),
		BindAddress:     envString("OFFLINE_MANAGER_BIND_ADDRESS", "0.0.0.0"),
		PublicHost:      envString("OFFLINE_MANAGER_PUBLIC_HOST", "127.0.0.1"),
		LogDir:          envString("OFFLINE_MANAGER_LOG_DIR", "logs/managers"),
		PortStart:       envInt("OFFLINE_MANAGER_PORT_START", 0),
		PortEnd:         envInt("OFFLINE_MANAGER_PORT_END", 0),
		GracefulTimeout: time.Duration(envInt("OFFLINE_MANAGER_GRACEFUL_TIMEOUT_SECONDS", 5)) * time.Second,
		Environment:     os.Environ(),
	}
	return cfg
}

type sessionProcessRuntime struct {
	cfg      SessionRuntimeConfig
	mu       sync.Mutex
	active   map[string]sessionProcess
	usedPort map[int]struct{}
}

type sessionProcess struct {
	session Session
	process *Process
}

// NewSessionRuntime 创建会话级 manager runtime。
func NewSessionRuntime(cfg SessionRuntimeConfig) SessionRuntime {
	if cfg.BinaryPath == "" {
		cfg.BinaryPath = defaultManagerBinaryPath()
	}
	if cfg.BindAddress == "" {
		cfg.BindAddress = "0.0.0.0"
	}
	if cfg.PublicHost == "" {
		cfg.PublicHost = "127.0.0.1"
	}
	if cfg.LogDir == "" {
		cfg.LogDir = "logs/managers"
	}
	if cfg.GracefulTimeout <= 0 {
		cfg.GracefulTimeout = 5 * time.Second
	}
	if len(cfg.Environment) == 0 {
		cfg.Environment = os.Environ()
	}

	return &sessionProcessRuntime{
		cfg:      cfg,
		active:   make(map[string]sessionProcess),
		usedPort: make(map[int]struct{}),
	}
}

func defaultManagerBinaryPath() string {
	name := fmt.Sprintf("gg20_sm_manager_%s_%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(".", "bin", name)
}

// NewSessionRuntimeFromEnv 创建使用环境变量配置的会话级 manager runtime。
func NewSessionRuntimeFromEnv() SessionRuntime {
	return NewSessionRuntime(DefaultSessionRuntimeConfig())
}

func (r *sessionProcessRuntime) StartSession(sessionKey string) (Session, error) {
	if strings.TrimSpace(sessionKey) == "" {
		return Session{}, fmt.Errorf("session_key 不能为空")
	}
	if _, err := os.Stat(r.cfg.BinaryPath); err != nil {
		return Session{}, fmt.Errorf("manager 二进制不可用: %w", err)
	}

	r.mu.Lock()
	if current, ok := r.active[sessionKey]; ok {
		r.mu.Unlock()
		return current.session, nil
	}

	port, err := r.allocatePortLocked()
	if err != nil {
		r.mu.Unlock()
		return Session{}, err
	}
	r.usedPort[port] = struct{}{}
	r.mu.Unlock()

	session := Session{
		SessionKey: sessionKey,
		ManagerURL: buildManagerURL(r.cfg.PublicHost, port),
		Room:       sessionKey,
		Port:       port,
	}

	logDir := filepath.Join(r.cfg.LogDir, sanitizePathPart(sessionKey))
	process := New(Config{
		BinaryPath:      r.cfg.BinaryPath,
		Args:            []string{"--address", r.cfg.BindAddress, "--port", strconv.Itoa(port)},
		LogDir:          logDir,
		AutoRestart:     false,
		GracefulTimeout: r.cfg.GracefulTimeout,
		Environment:     r.cfg.Environment,
	})
	if err := process.Start(); err != nil {
		r.releasePort(port)
		return Session{}, err
	}

	r.mu.Lock()
	r.active[sessionKey] = sessionProcess{session: session, process: process}
	r.mu.Unlock()
	return session, nil
}

func (r *sessionProcessRuntime) StopSession(sessionKey string) error {
	r.mu.Lock()
	current, ok := r.active[sessionKey]
	if ok {
		delete(r.active, sessionKey)
		delete(r.usedPort, current.session.Port)
	}
	r.mu.Unlock()
	if !ok {
		return nil
	}
	return current.process.Stop()
}

func (r *sessionProcessRuntime) StopAll() error {
	r.mu.Lock()
	processes := make([]sessionProcess, 0, len(r.active))
	for sessionKey, current := range r.active {
		processes = append(processes, current)
		delete(r.active, sessionKey)
		delete(r.usedPort, current.session.Port)
	}
	r.mu.Unlock()

	var firstErr error
	for _, current := range processes {
		if err := current.process.Stop(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (r *sessionProcessRuntime) allocatePortLocked() (int, error) {
	if r.cfg.PortStart > 0 && r.cfg.PortEnd >= r.cfg.PortStart {
		for port := r.cfg.PortStart; port <= r.cfg.PortEnd; port++ {
			if _, used := r.usedPort[port]; used {
				continue
			}
			if portAvailable(r.cfg.BindAddress, port) {
				return port, nil
			}
		}
		return 0, fmt.Errorf("没有可用 manager 端口: %d-%d", r.cfg.PortStart, r.cfg.PortEnd)
	}

	listener, err := net.Listen("tcp", net.JoinHostPort(r.cfg.BindAddress, "0"))
	if err != nil {
		return 0, fmt.Errorf("分配 manager 端口失败: %w", err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port
	if _, used := r.usedPort[port]; used {
		return 0, fmt.Errorf("分配到重复 manager 端口: %d", port)
	}
	return port, nil
}

func (r *sessionProcessRuntime) releasePort(port int) {
	r.mu.Lock()
	delete(r.usedPort, port)
	r.mu.Unlock()
}

func portAvailable(bindAddress string, port int) bool {
	listener, err := net.Listen("tcp", net.JoinHostPort(bindAddress, strconv.Itoa(port)))
	if err != nil {
		return false
	}
	_ = listener.Close()
	return true
}

func buildManagerURL(publicHost string, port int) string {
	raw := strings.TrimSpace(publicHost)
	if raw == "" {
		raw = "127.0.0.1"
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		return "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	}
	parsed.Host = net.JoinHostPort(parsed.Hostname(), strconv.Itoa(port))
	parsed.Path = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/")
}

func sanitizePathPart(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "session"
	}
	return b.String()
}

func envString(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
