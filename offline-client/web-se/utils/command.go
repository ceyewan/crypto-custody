package utils

import (
	"bytes"
	"context"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"web-se/clog"
	"web-se/config"
)

const (
	defaultTimeout = 60 * time.Second
)

// ExecCommand 执行命令并返回输出
func ExecCommand(ctx context.Context, cfg *config.Config, name string, args ...string) (string, error) {
	// 设置超时上下文
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	// 准备命令
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 记录开始执行
	logCommandStart(name, args)

	// 执行命令并计时
	startTime := time.Now()
	err := cmd.Run()
	executionTime := time.Since(startTime)

	// 处理执行结果
	if err != nil {
		logCommandFailure(err, stdout.String(), stderr.String(), executionTime)
		return stderr.String(), err
	}

	logCommandSuccess(stdout.String(), executionTime)
	return stdout.String(), nil
}

// RunKeyGen 运行密钥生成命令
func RunKeyGen(ctx context.Context, cfg *config.Config, t, n, i int, output string) error {
	cmdPath := filepath.Join(cfg.BinDir, cfg.KeygenBin)

	args := buildKeygenArgs(cfg, t, n, i, output)

	clog.Info("开始密钥生成",
		clog.String("command", cmdPath),
		clog.Int("threshold", t),
		clog.Int("parties", n),
		clog.Int("index", i),
		clog.String("output", output))

	_, err := ExecCommand(ctx, cfg, cmdPath, args...)
	if err != nil {
		clog.Error("密钥生成失败", clog.Err(err))
		return err
	}

	clog.Info("密钥生成成功")
	return nil
}

// RunSigning 运行签名命令
func RunSigning(ctx context.Context, cfg *config.Config, parties, data, localShare string) (string, error) {
	cmdPath := filepath.Join(cfg.BinDir, cfg.SigningBin)

	args := buildSigningArgs(cfg, parties, data, localShare)

	clog.Info("开始签名操作",
		clog.String("command", cmdPath),
		clog.String("parties", parties),
		clog.String("data", data),
		clog.String("local_share", localShare))

	output, err := ExecCommand(ctx, cfg, cmdPath, args...)
	if err != nil {
		clog.Error("签名操作失败", clog.Err(err))
		return "", err
	}

	clog.Info("签名操作成功")
	return output, nil
}

// === 辅助函数 ===

// 将值转换为字符串
func toString(value interface{}) string {
	switch v := value.(type) {
	case int:
		return strconv.Itoa(v)
	case string:
		return v
	default:
		return ""
	}
}

// 构建密钥生成命令的参数
func buildKeygenArgs(cfg *config.Config, t, n, i int, output string) []string {
	return []string{
		// "--address", cfg.ManagerAddr,
		"--threshold", toString(t),
		"--number-of-parties", toString(n),
		"--index", toString(i),
		"--output", output,
	}
}

// 构建签名命令的参数
func buildSigningArgs(cfg *config.Config, parties, data, localShare string) []string {
	return []string{
		// "--address", cfg.ManagerAddr,
		"--parties", parties,
		"--data-to-sign", data,
		"--local-share", localShare,
	}
}

// 记录命令开始执行
func logCommandStart(name string, args []string) {
	clog.Info("开始执行命令",
		clog.String("command", name),
		clog.String("args", strings.Join(args, " ")),
		clog.String("timeout", defaultTimeout.String()))
}

// 记录命令执行成功
func logCommandSuccess(stdout string, executionTime time.Duration) {
	clog.Info("命令执行成功",
		clog.String("stdout", stdout),
		clog.String("execution_time", executionTime.String()))
}

// 记录命令执行失败
func logCommandFailure(err error, stdout, stderr string, executionTime time.Duration) {
	clog.Error("命令执行失败",
		clog.Err(err),
		clog.String("stdout", stdout),
		clog.String("stderr", stderr),
		clog.String("execution_time", executionTime.String()))
}
