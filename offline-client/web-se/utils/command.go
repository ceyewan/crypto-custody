package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"web-se/config"
)

// ExecCommand 执行命令并返回输出
func ExecCommand(ctx context.Context, cfg *config.Config, name string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer

	// 设置命令
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// 记录命令执行信息
	LogInfo("开始执行命令",
		String("command", name),
		String("args", strings.Join(args, " ")),
		String("timeout", "60s"))

	// 执行命令
	startTime := time.Now()
	err := cmd.Run()
	executionTime := time.Since(startTime)

	if err != nil {
		// 记录错误信息
		LogError("命令执行失败",
			Error(err),
			String("stdout", stdout.String()),
			String("stderr", stderr.String()),
			String("execution_time", executionTime.String()))
		return stderr.String(), err
	}

	// 记录成功信息
	LogInfo("命令执行成功",
		String("stdout", stdout.String()),
		String("execution_time", executionTime.String()))

	return stdout.String(), nil
}

// RunKeyGen 运行密钥生成命令
func RunKeyGen(ctx context.Context, cfg *config.Config, t, n, i int, output string) error {
	// 构建命令路径
	cmdPath := filepath.Join(cfg.BinDir, cfg.KeygenBin)

	// 构建命令参数
	args := []string{
		"--address", cfg.ManagerAddr,
		"--index", toString(i),
		"--number-of-parties", toString(n),
		"--output", output,
		"--threshold", toString(t),
		"--room", "default",
	}

	// 记录密钥生成信息
	LogInfo("开始密钥生成",
		String("command", cmdPath),
		Int("threshold", t),
		Int("parties", n),
		Int("index", i),
		String("output", output))

	// 执行命令
	_, err := ExecCommand(ctx, cfg, cmdPath, args...)
	if err != nil {
		LogError("密钥生成失败", Error(err))
		return err
	}

	LogInfo("密钥生成成功")
	return nil
}

// RunSigning 运行签名命令
func RunSigning(ctx context.Context, cfg *config.Config, parties, data, localShare string) (string, error) {
	// 构建命令路径
	cmdPath := filepath.Join(cfg.BinDir, cfg.SigningBin)

	// 构建命令参数
	args := []string{
		"-p", parties,
		"-d", data,
		"-l", localShare,
	}

	// 记录签名信息
	LogInfo("开始签名操作",
		String("command", cmdPath),
		String("parties", parties),
		String("data", data),
		String("local_share", localShare))

	// 执行命令
	output, err := ExecCommand(ctx, cfg, cmdPath, args...)
	if err != nil {
		LogError("签名操作失败", Error(err))
		return "", err
	}

	LogInfo("签名操作成功")
	return output, nil
}

// toString 将各种类型转换为字符串
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

// ParseJSONFile 解析JSON文件并提取公钥
func ParseJSONFile(filePath string) (map[string]interface{}, error) {
	// 读取文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	// 解析JSON
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// ExtractPublicKey 从JSON数据中提取公钥
func ExtractPublicKey(jsonData map[string]interface{}) (string, error) {
	// 检查y_sum_s字段是否存在
	ySumS, ok := jsonData["y_sum_s"]
	if !ok {
		return "", errors.New("找不到y_sum_s字段")
	}

	// 将y_sum_s转换为map
	ySumSMap, ok := ySumS.(map[string]interface{})
	if !ok {
		return "", errors.New("y_sum_s字段格式不正确")
	}

	// 检查point字段是否存在
	point, ok := ySumSMap["point"]
	if !ok {
		return "", errors.New("找不到point字段")
	}

	// 将point转换为[]interface{}
	pointArray, ok := point.([]interface{})
	if !ok {
		return "", errors.New("point字段不是数组")
	}

	// 将point数组转换为字符串（例如十六进制格式）
	var publicKeyParts []string
	for _, value := range pointArray {
		intValue, ok := value.(float64) // JSON解析时数字会被解析为float64
		if !ok {
			return "", errors.New("point数组包含非数字元素")
		}
		publicKeyParts = append(publicKeyParts, fmt.Sprintf("%02x", int(intValue)))
	}

	// 拼接为最终的公钥字符串
	publicKey := strings.Join(publicKeyParts, "")
	return publicKey, nil
}
