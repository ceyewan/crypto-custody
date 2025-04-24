package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
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

	// 执行命令
	err := cmd.Run()
	if err != nil {
		// 返回stderr输出和错误
		return stderr.String(), err
	}

	return stdout.String(), nil
}

// RunKeyGen 运行密钥生成命令
func RunKeyGen(ctx context.Context, cfg *config.Config, t, n, i int, output string) error {
	// 构建命令路径
	cmdPath := filepath.Join(cfg.BinDir, cfg.KeygenBin)

	// 构建命令参数
	args := []string{
		"--address", cfg.ManagerAddr,
		"-t", toString(t),
		"-n", toString(n),
		"-i", toString(i),
		"--output", output,
	}

	// 执行命令
	_, err := ExecCommand(ctx, cfg, cmdPath, args...)
	return err
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

	// 执行命令
	output, err := ExecCommand(ctx, cfg, cmdPath, args...)
	if err != nil {
		return "", err
	}

	return output, nil
}

// toString 将各种类型转换为字符串
func toString(value interface{}) string {
	switch v := value.(type) {
	case int:
		return strings.TrimSpace(strings.Replace(string(append([]byte{}, byte(v))), "\x00", "", -1))
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

	// 将point转换为字符串
	publicKey, ok := point.(string)
	if !ok {
		return "", errors.New("point字段不是字符串")
	}

	return publicKey, nil
}
