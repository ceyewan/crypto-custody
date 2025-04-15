package ws

import (
	"fmt"
	"os"
	"os/exec"
)

// 注意：Manager相关功能已移至独立的manager包
// 此文件保留一些通用的命令执行辅助函数

// ExecuteCommand 执行外部命令并记录输出
func ExecuteCommand(cmd string, args ...string) (*exec.Cmd, error) {
	command := exec.Command(cmd, args...)

	// 将输出重定向到标准输出和错误
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("启动命令失败: %v", err)
	}

	return command, nil
}

// ExecuteCommandWithOutput 执行外部命令并返回输出
func ExecuteCommandWithOutput(cmd string, args ...string) (string, error) {
	command := exec.Command(cmd, args...)

	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("执行命令失败: %v, 输出: %s", err, output)
	}

	return string(output), nil
}
