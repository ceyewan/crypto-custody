package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"web-se/clog"

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
	clog.Info("创建MPC服务")
	return &MPCService{
		cfg:             cfg,
		securityService: securityService,
	}
}

// KeyGeneration 密钥生成
func (s *MPCService) KeyGeneration(ctx context.Context, threshold, parties, index int, filename, userName string) (string, []byte, error) {
	clog.Debug("开始密钥生成过程",
		clog.Int("threshold", threshold),
		clog.Int("parties", parties),
		clog.Int("index", index),
		clog.String("filename", filename))

	// 确保临时目录存在
	if err := utils.EnsureDir(s.cfg.TempDir); err != nil {
		clog.Error("创建临时目录失败", clog.Err(err), clog.String("dir", s.cfg.TempDir))
		return "", nil, fmt.Errorf("创建临时目录失败: %v", err)
	}

	// 构建文件路径
	filePath := filepath.Join(s.cfg.TempDir, filename)
	clog.Debug("临时文件路径", clog.String("path", filePath))

	// 清理函数（在函数返回前删除临时文件）
	defer func() {
		clog.Debug("清理临时文件", clog.String("path", filePath))
		if err := utils.DeleteFile(filePath); err != nil {
			clog.Warn("删除临时文件失败", clog.Err(err), clog.String("path", filePath))
		}
	}()

	// 运行密钥生成命令
	clog.Info("执行密钥生成命令")
	if err := utils.RunKeyGen(ctx, s.cfg, threshold, parties, index, filePath); err != nil {
		clog.Error("密钥生成命令失败", clog.Err(err))
		return "", nil, fmt.Errorf("密钥生成失败: %v", err)
	}

	// 解析生成的JSON文件
	clog.Debug("解析JSON文件", clog.String("path", filePath))
	jsonData, err := utils.ParseJSONFile(filePath)
	if err != nil {
		clog.Error("解析JSON文件失败", clog.Err(err))
		return "", nil, fmt.Errorf("解析JSON文件失败: %v", err)
	}

	// 从JSON提取公钥
	clog.Debug("从JSON提取公钥")
	publicKey, err := utils.ExtractPublicKey(jsonData)
	if err != nil {
		clog.Error("提取公钥失败", clog.Err(err))
		return "", nil, fmt.Errorf("提取公钥失败: %v", err)
	}

	// 从公钥提取以太坊地址
	clog.Debug("从公钥提取以太坊地址", clog.String("publicKey", publicKey))
	address, err := utils.ExtractEthAddress(publicKey)
	if err != nil {
		clog.Error("提取地址失败", clog.Err(err))
		return "", nil, fmt.Errorf("提取地址失败: %v", err)
	}
	// 确保地址格式正确
	if !strings.HasPrefix(address, "0x") {
		address = "0x" + address
	}
	clog.Info("生成以太坊地址", clog.String("address", address))

	// 读取文件内容
	clog.Debug("读取生成的密钥文件")
	fileContent, err := utils.ReadFile(filePath)
	if err != nil {
		clog.Error("读取JSON文件失败", clog.Err(err))
		return "", nil, fmt.Errorf("读取JSON文件失败: %v", err)
	}
	clog.Debug("文件内容大小", clog.String("size", utils.FormatByteSize(int64(len(fileContent)))))

	// 压缩文件内容
	clog.Debug("压缩文件内容")
	compressedData, err := utils.CompressData(fileContent)
	if err != nil {
		clog.Error("压缩数据失败", clog.Err(err))
		return "", nil, fmt.Errorf("压缩数据失败: %v", err)
	}
	clog.Debug("压缩后数据大小", clog.String("size", utils.FormatByteSize(int64(len(compressedData)))))

	// 生成32字节的随机数
	clog.Debug("生成随机加密密钥")
	randomKey, err := utils.GenerateRandomBytes(32)
	if err != nil {
		clog.Error("生成随机密钥失败", clog.Err(err))
		return "", nil, fmt.Errorf("生成随机密钥失败: %v", err)
	}

	// 使用随机数加密压缩后的JSON文件
	clog.Debug("加密压缩数据")
	encryptedData, err := utils.EncryptAES(compressedData, randomKey)
	if err != nil {
		clog.Error("加密数据失败", clog.Err(err))
		return "", nil, fmt.Errorf("加密数据失败: %v", err)
	}
	clog.Debug("加密后数据大小", clog.String("size", utils.FormatByteSize(int64(len(encryptedData)))))

	// 将随机数存储到安全芯片
	clog.Info("存储密钥到安全芯片",
		clog.String("userName", userName),
		clog.String("address", address),
		clog.String("message", string(randomKey)))
	if err := s.securityService.StoreData(userName, address, randomKey); err != nil {
		clog.Error("存储密钥到安全芯片失败", clog.Err(err))
		return "", nil, fmt.Errorf("存储密钥到安全芯片失败: %v", err)
	}

	clog.Info("密钥生成完成")
	return address, encryptedData, nil
}

// SignMessage 消息签名
func (s *MPCService) SignMessage(ctx context.Context, parties, data, filename, userName, address string, encryptedKey, signature []byte) (string, error) {
	clog.Debug("开始消息签名过程",
		clog.String("parties", parties),
		clog.String("address", address),
		clog.String("filename", filename))

	// 检查data是否有0x前缀，如果有则移除
	if strings.HasPrefix(data, "0x") {
		data = data[2:]
		clog.Debug("移除数据0x前缀", clog.String("data", data))
	}

	// 确保临时目录存在
	if err := utils.EnsureDir(s.cfg.TempDir); err != nil {
		clog.Error("创建临时目录失败", clog.Err(err), clog.String("dir", s.cfg.TempDir))
		return "", fmt.Errorf("创建临时目录失败: %v", err)
	}

	// 构建文件路径
	filePath := filepath.Join(s.cfg.TempDir, filename)
	clog.Debug("临时文件路径", clog.String("path", filePath))

	// 清理函数（在函数返回前删除临时文件）
	defer func() {
		// clog.Debug("清理临时文件", clog.String("path", filePath))
		// if err := utils.DeleteFile(filePath); err != nil {
		// 	clog.Warn("删除临时文件失败", clog.Err(err), clog.String("path", filePath))
		// }
	}()

	// 从安全芯片读取随机数
	clog.Info("从安全芯片读取密钥", clog.String("userName", userName), clog.String("address", address))
	randomKey, err := s.securityService.ReadData(userName, address, signature)
	if err != nil {
		clog.Error("从安全芯片读取密钥失败", clog.Err(err))
		return "", fmt.Errorf("从安全芯片读取密钥失败: %v", err)
	}

	// 使用随机数解密数据
	clog.Debug("解密数据")
	decryptedData, err := utils.DecryptAES(encryptedKey, randomKey)
	if err != nil {
		clog.Error("解密数据失败", clog.Err(err))
		return "", fmt.Errorf("解密数据失败: %v", err)
	}
	clog.Debug("解密后数据大小", clog.String("size", utils.FormatByteSize(int64(len(decryptedData)))))

	// 解压数据
	clog.Debug("解压数据")
	decompressedData, err := utils.DecompressData(decryptedData)
	if err != nil {
		clog.Error("解压数据失败", clog.Err(err))
		return "", fmt.Errorf("解压数据失败: %v", err)
	}
	clog.Debug("解压后数据大小", clog.String("size", utils.FormatByteSize(int64(len(decompressedData)))))

	// 将解密后的数据写入临时文件
	clog.Debug("写入临时文件", clog.String("path", filePath))
	if err := utils.WriteFile(filePath, decompressedData); err != nil {
		clog.Error("写入临时文件失败", clog.Err(err))
		return "", fmt.Errorf("写入临时文件失败: %v", err)
	}

	// 运行签名命令
	clog.Info("执行签名命令",
		clog.String("parties", parties),
		clog.String("data", data))
	signResult, err := utils.RunSigning(ctx, s.cfg, parties, data, filePath)
	if err != nil {
		clog.Error("签名命令失败", clog.Err(err))
		return "", fmt.Errorf("签名失败: %v", err)
	}

	// 格式化签名结果
	signResult = strings.TrimSpace(signResult)
	if !strings.HasPrefix(signResult, "0x") {
		signResult = "0x" + signResult
	}

	clog.Info("签名完成", clog.String("signature", signResult))
	return signResult, nil
}
