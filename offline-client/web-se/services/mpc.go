package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

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
	utils.LogInfo("创建MPC服务")
	return &MPCService{
		cfg:             cfg,
		securityService: securityService,
	}
}

// KeyGeneration 密钥生成
func (s *MPCService) KeyGeneration(ctx context.Context, threshold, parties, index int, filename, userName string) (string, []byte, error) {
	utils.LogDebug("开始密钥生成过程",
		utils.Int("threshold", threshold),
		utils.Int("parties", parties),
		utils.Int("index", index),
		utils.String("filename", filename))

	// 确保临时目录存在
	if err := utils.EnsureDir(s.cfg.TempDir); err != nil {
		utils.LogError("创建临时目录失败", utils.Error(err), utils.String("dir", s.cfg.TempDir))
		return "", nil, fmt.Errorf("创建临时目录失败: %v", err)
	}

	// 构建文件路径
	filePath := filepath.Join(s.cfg.TempDir, filename)
	utils.LogDebug("临时文件路径", utils.String("path", filePath))

	// 清理函数（在函数返回前删除临时文件）
	defer func() {
		utils.LogDebug("清理临时文件", utils.String("path", filePath))
		if err := utils.DeleteFile(filePath); err != nil {
			utils.LogWarn("删除临时文件失败", utils.Error(err), utils.String("path", filePath))
		}
	}()

	// 运行密钥生成命令
	utils.LogInfo("执行密钥生成命令")
	if err := utils.RunKeyGen(ctx, s.cfg, threshold, parties, index, filePath); err != nil {
		utils.LogError("密钥生成命令失败", utils.Error(err))
		return "", nil, fmt.Errorf("密钥生成失败: %v", err)
	}

	// 解析生成的JSON文件
	utils.LogDebug("解析JSON文件", utils.String("path", filePath))
	jsonData, err := utils.ParseJSONFile(filePath)
	if err != nil {
		utils.LogError("解析JSON文件失败", utils.Error(err))
		return "", nil, fmt.Errorf("解析JSON文件失败: %v", err)
	}

	// 从JSON提取公钥
	utils.LogDebug("从JSON提取公钥")
	publicKey, err := utils.ExtractPublicKey(jsonData)
	if err != nil {
		utils.LogError("提取公钥失败", utils.Error(err))
		return "", nil, fmt.Errorf("提取公钥失败: %v", err)
	}

	// 从公钥提取以太坊地址
	utils.LogDebug("从公钥提取以太坊地址", utils.String("publicKey", publicKey))
	address, err := utils.ExtractEthAddress(publicKey)
	if err != nil {
		utils.LogError("提取地址失败", utils.Error(err))
		return "", nil, fmt.Errorf("提取地址失败: %v", err)
	}
	// 确保地址格式正确
	if !strings.HasPrefix(address, "0x") {
		address = "0x" + address
	}
	utils.LogInfo("生成以太坊地址", utils.String("address", address))

	// 读取文件内容
	utils.LogDebug("读取生成的密钥文件")
	fileContent, err := utils.ReadFile(filePath)
	if err != nil {
		utils.LogError("读取JSON文件失败", utils.Error(err))
		return "", nil, fmt.Errorf("读取JSON文件失败: %v", err)
	}
	utils.LogDebug("文件内容大小", utils.String("size", utils.FormatByteSize(int64(len(fileContent)))))

	// 压缩文件内容
	utils.LogDebug("压缩文件内容")
	compressedData, err := utils.CompressData(fileContent)
	if err != nil {
		utils.LogError("压缩数据失败", utils.Error(err))
		return "", nil, fmt.Errorf("压缩数据失败: %v", err)
	}
	utils.LogDebug("压缩后数据大小", utils.String("size", utils.FormatByteSize(int64(len(compressedData)))))

	// 生成32字节的随机数
	utils.LogDebug("生成随机加密密钥")
	randomKey, err := utils.GenerateRandomBytes(32)
	if err != nil {
		utils.LogError("生成随机密钥失败", utils.Error(err))
		return "", nil, fmt.Errorf("生成随机密钥失败: %v", err)
	}

	// 使用随机数加密压缩后的JSON文件
	utils.LogDebug("加密压缩数据")
	encryptedData, err := utils.EncryptAES(compressedData, randomKey)
	if err != nil {
		utils.LogError("加密数据失败", utils.Error(err))
		return "", nil, fmt.Errorf("加密数据失败: %v", err)
	}
	utils.LogDebug("加密后数据大小", utils.String("size", utils.FormatByteSize(int64(len(encryptedData)))))

	// 将随机数存储到安全芯片
	utils.LogInfo("存储密钥到安全芯片",
		utils.String("userName", userName),
		utils.String("address", address))
	if err := s.securityService.StoreData(userName, address, randomKey); err != nil {
		utils.LogError("存储密钥到安全芯片失败", utils.Error(err))
		return "", nil, fmt.Errorf("存储密钥到安全芯片失败: %v", err)
	}

	utils.LogInfo("密钥生成完成")
	return address, encryptedData, nil
}

// SignMessage 消息签名
func (s *MPCService) SignMessage(ctx context.Context, parties, data, filename, userName, address string, encryptedKey, signature []byte) (string, error) {
	utils.LogDebug("开始消息签名过程",
		utils.String("parties", parties),
		utils.String("address", address),
		utils.String("filename", filename))

	// 检查data是否有0x前缀，如果有则移除
	if strings.HasPrefix(data, "0x") {
		data = data[2:]
		utils.LogDebug("移除数据0x前缀", utils.String("data", data))
	}

	// 确保临时目录存在
	if err := utils.EnsureDir(s.cfg.TempDir); err != nil {
		utils.LogError("创建临时目录失败", utils.Error(err), utils.String("dir", s.cfg.TempDir))
		return "", fmt.Errorf("创建临时目录失败: %v", err)
	}

	// 构建文件路径
	filePath := filepath.Join(s.cfg.TempDir, filename)
	utils.LogDebug("临时文件路径", utils.String("path", filePath))

	// 清理函数（在函数返回前删除临时文件）
	defer func() {
		utils.LogDebug("清理临时文件", utils.String("path", filePath))
		if err := utils.DeleteFile(filePath); err != nil {
			utils.LogWarn("删除临时文件失败", utils.Error(err), utils.String("path", filePath))
		}
	}()

	// 从安全芯片读取随机数
	utils.LogInfo("从安全芯片读取密钥", utils.String("userName", userName), utils.String("address", address))
	randomKey, err := s.securityService.ReadData(userName, address, signature)
	if err != nil {
		utils.LogError("从安全芯片读取密钥失败", utils.Error(err))
		return "", fmt.Errorf("从安全芯片读取密钥失败: %v", err)
	}

	// 使用随机数解密数据
	utils.LogDebug("解密数据")
	decryptedData, err := utils.DecryptAES(encryptedKey, randomKey)
	if err != nil {
		utils.LogError("解密数据失败", utils.Error(err))
		return "", fmt.Errorf("解密数据失败: %v", err)
	}
	utils.LogDebug("解密后数据大小", utils.String("size", utils.FormatByteSize(int64(len(decryptedData)))))

	// 解压数据
	utils.LogDebug("解压数据")
	decompressedData, err := utils.DecompressData(decryptedData)
	if err != nil {
		utils.LogError("解压数据失败", utils.Error(err))
		return "", fmt.Errorf("解压数据失败: %v", err)
	}
	utils.LogDebug("解压后数据大小", utils.String("size", utils.FormatByteSize(int64(len(decompressedData)))))

	// 将解密后的数据写入临时文件
	utils.LogDebug("写入临时文件", utils.String("path", filePath))
	if err := utils.WriteFile(filePath, decompressedData); err != nil {
		utils.LogError("写入临时文件失败", utils.Error(err))
		return "", fmt.Errorf("写入临时文件失败: %v", err)
	}

	// 运行签名命令
	utils.LogInfo("执行签名命令",
		utils.String("parties", parties),
		utils.String("data", data))
	signResult, err := utils.RunSigning(ctx, s.cfg, parties, data, filePath)
	if err != nil {
		utils.LogError("签名命令失败", utils.Error(err))
		return "", fmt.Errorf("签名失败: %v", err)
	}

	// 格式化签名结果
	signResult = strings.TrimSpace(signResult)
	if !strings.HasPrefix(signResult, "0x") {
		signResult = "0x" + signResult
	}

	utils.LogInfo("签名完成", utils.String("signature", signResult))
	return signResult, nil
}
