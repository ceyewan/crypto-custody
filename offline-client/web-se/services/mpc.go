package services

import (
	"context"
	"fmt"
	"path/filepath"

	"web-se/config"
	"web-se/utils"
)

// MPCService MPC服务
type MPCService struct {
	cfg             *config.Config
	securityService *SecurityService
}

// NewMPCService 创建MPC服务
func NewMPCService(cfg *config.Config, securityService *SecurityService) *MPCService {
	return &MPCService{
		cfg:             cfg,
		securityService: securityService,
	}
}

// KeyGeneration 密钥生成
func (s *MPCService) KeyGeneration(ctx context.Context, threshold, parties, index int, filename, userName string) (string, []byte, error) {
	// 确保临时目录存在
	if err := utils.EnsureDir(s.cfg.TempDir); err != nil {
		return "", nil, fmt.Errorf("创建临时目录失败: %v", err)
	}

	// 构建文件路径
	filePath := filepath.Join(s.cfg.TempDir, filename)

	// 清理函数（在函数返回前删除临时文件）
	defer func() {
		if err := utils.DeleteFile(filePath); err != nil {
			fmt.Printf("删除临时文件 %s 失败: %v\n", filePath, err)
		}
	}()

	// 运行密钥生成命令
	if err := utils.RunKeyGen(ctx, s.cfg, threshold, parties, index, filePath); err != nil {
		return "", nil, fmt.Errorf("密钥生成失败: %v", err)
	}

	// 解析生成的JSON文件
	jsonData, err := utils.ParseJSONFile(filePath)
	if err != nil {
		return "", nil, fmt.Errorf("解析JSON文件失败: %v", err)
	}

	// 从JSON提取公钥
	publicKey, err := utils.ExtractPublicKey(jsonData)
	if err != nil {
		return "", nil, fmt.Errorf("提取公钥失败: %v", err)
	}

	// 从公钥提取以太坊地址
	address, err := utils.ExtractEthAddress(publicKey)
	if err != nil {
		return "", nil, fmt.Errorf("提取地址失败: %v", err)
	}

	// 读取文件内容
	fileContent, err := utils.ReadFile(filePath)
	if err != nil {
		return "", nil, fmt.Errorf("读取JSON文件失败: %v", err)
	}

	// 压缩文件内容
	compressedData, err := utils.CompressData(fileContent)
	if err != nil {
		return "", nil, fmt.Errorf("压缩数据失败: %v", err)
	}

	// 生成32字节的随机数
	randomKey, err := utils.GenerateRandomBytes(32)
	if err != nil {
		return "", nil, fmt.Errorf("生成随机密钥失败: %v", err)
	}

	// 使用随机数加密压缩后的JSON文件
	encryptedData, err := utils.EncryptAES(compressedData, randomKey)
	if err != nil {
		return "", nil, fmt.Errorf("加密数据失败: %v", err)
	}

	// 将随机数存储到安全芯片
	if err := s.securityService.StoreData(userName, address, randomKey); err != nil {
		return "", nil, fmt.Errorf("存储密钥到安全芯片失败: %v", err)
	}

	return address, encryptedData, nil
}

// SignMessage 消息签名
func (s *MPCService) SignMessage(ctx context.Context, parties, data, filename, userName, address string, encryptedKey, signature []byte) (string, error) {
	// 确保临时目录存在
	if err := utils.EnsureDir(s.cfg.TempDir); err != nil {
		return "", fmt.Errorf("创建临时目录失败: %v", err)
	}

	// 构建文件路径
	filePath := filepath.Join(s.cfg.TempDir, filename)

	// 清理函数（在函数返回前删除临时文件）
	defer func() {
		if err := utils.DeleteFile(filePath); err != nil {
			fmt.Printf("删除临时文件 %s 失败: %v\n", filePath, err)
		}
	}()

	// 从安全芯片读取随机数
	randomKey, err := s.securityService.ReadData(userName, address, signature)
	if err != nil {
		return "", fmt.Errorf("从安全芯片读取密钥失败: %v", err)
	}

	// 使用随机数解密数据
	decryptedData, err := utils.DecryptAES(encryptedKey, randomKey)
	if err != nil {
		return "", fmt.Errorf("解密数据失败: %v", err)
	}

	// 解压数据
	decompressedData, err := utils.DecompressData(decryptedData)
	if err != nil {
		return "", fmt.Errorf("解压数据失败: %v", err)
	}

	// 将解密后的数据写入临时文件
	if err := utils.WriteFile(filePath, decompressedData); err != nil {
		return "", fmt.Errorf("写入临时文件失败: %v", err)
	}

	// 运行签名命令
	signResult, err := utils.RunSigning(ctx, s.cfg, parties, data, filePath)
	if err != nil {
		return "", fmt.Errorf("签名失败: %v", err)
	}

	return signResult, nil
}
