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

// MPCService 提供多方计算(MPC)相关功能的服务接口。
// 它处理密钥生成、消息签名等与MPC相关的操作。
type MPCService struct {
	cfg             *config.Config
	securityService *SecurityService
}

// NewMPCService 创建并初始化MPC服务实例。
// 参数:
//   - cfg: 系统配置对象
//   - securityService: 安全服务实例，用于进行安全芯片操作
//
// 返回:
//   - *MPCService: 初始化后的服务实例
func NewMPCService(cfg *config.Config, securityService *SecurityService) *MPCService {
	clog.Debug("初始化MPC服务")
	return &MPCService{
		cfg:             cfg,
		securityService: securityService,
	}
}

// KeyGeneration 执行MPC密钥生成流程。
// 生成密钥后，加密存储到安全芯片中。
//
// 参数:
//   - ctx: 上下文，用于控制操作超时或取消
//   - threshold: 签名阈值，表示需要多少方参与才能完成签名
//   - parties: 参与方总数
//   - index: 当前节点索引
//   - filename: 临时文件名
//   - userName: 用户名，用于安全芯片存储
//
// 返回:
//   - string: 生成的以太坊地址
//   - []byte: 加密后的密钥数据
//   - error: 操作过程中遇到的错误
func (s *MPCService) KeyGeneration(ctx context.Context, threshold, parties, index int, filename, userName string) (string, []byte, error) {
	clog.Debug("开始密钥生成",
		clog.Int("threshold", threshold),
		clog.Int("parties", parties),
		clog.Int("index", index),
		clog.String("file", filename))

	// 确保临时目录存在
	if err := utils.EnsureDir(s.cfg.TempDir); err != nil {
		clog.Error("创建临时目录失败", clog.Err(err))
		return "", nil, fmt.Errorf("创建临时目录失败: %w", err)
	}

	// 构建文件路径
	filePath := filepath.Join(s.cfg.TempDir, filename)
	clog.Debug("临时文件路径", clog.String("path", filePath))

	// 清理函数（在函数返回前删除临时文件）
	defer func() {
		if err := utils.DeleteFile(filePath); err != nil {
			clog.Warn("清理临时文件失败", clog.Err(err))
		}
	}()

	// 运行密钥生成命令
	clog.Info("执行密钥生成")
	if err := utils.RunKeyGen(ctx, s.cfg, threshold, parties, index, filePath); err != nil {
		clog.Error("密钥生成失败", clog.Err(err))
		return "", nil, fmt.Errorf("密钥生成失败: %w", err)
	}

	// 解析生成的JSON文件
	clog.Info("解析生成的密钥文件")
	jsonData, err := utils.ParseJSONFile(filePath)
	if err != nil {
		clog.Error("解析JSON失败", clog.Err(err))
		return "", nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	// 从JSON提取公钥
	publicKey, err := utils.ExtractPublicKey(jsonData)
	if err != nil {
		clog.Error("提取公钥失败", clog.Err(err))
		return "", nil, fmt.Errorf("提取公钥失败: %w", err)
	}
	clog.Debug("提取公钥", clog.String("pubkey", publicKey))

	// 从公钥提取以太坊地址
	address, err := utils.ExtractEthAddress(publicKey)
	if err != nil {
		clog.Error("提取地址失败", clog.Err(err))
		return "", nil, fmt.Errorf("提取地址失败: %w", err)
	}

	// 确保地址格式正确
	if !strings.HasPrefix(address, "0x") {
		address = "0x" + address
	}
	clog.Info("生成以太坊地址", clog.String("addr", address))

	// 读取文件内容
	fileContent, err := utils.ReadFile(filePath)
	if err != nil {
		clog.Error("读取密钥文件失败", clog.Err(err))
		return "", nil, fmt.Errorf("读取密钥文件失败: %w", err)
	}
	clog.Debug("密钥文件大小", clog.Int("size", len(fileContent)))

	// 压缩文件内容
	compressedData, err := utils.CompressData(fileContent)
	if err != nil {
		clog.Error("压缩数据失败", clog.Err(err))
		return "", nil, fmt.Errorf("压缩数据失败: %w", err)
	}
	clog.Debug("压缩后数据大小", clog.Int("size", len(compressedData)))

	// 生成32字节的随机数作为加密密钥
	randomKey, err := utils.GenerateRandomBytes(32)
	if err != nil {
		clog.Error("生成随机密钥失败", clog.Err(err))
		return "", nil, fmt.Errorf("生成随机密钥失败: %w", err)
	}

	// 使用随机数加密压缩后的JSON文件
	clog.Info("加密密钥数据")
	encryptedData, err := utils.EncryptAES(compressedData, randomKey)
	if err != nil {
		clog.Error("加密数据失败", clog.Err(err))
		return "", nil, fmt.Errorf("加密数据失败: %w", err)
	}
	clog.Debug("加密后数据大小", clog.Int("size", len(encryptedData)))

	// 将随机数存储到安全芯片
	clog.Info("存储密钥到安全芯片")
	if err := s.securityService.StoreData(userName, address, randomKey); err != nil {
		clog.Error("存储密钥失败", clog.Err(err))
		return "", nil, fmt.Errorf("存储密钥到安全芯片失败: %w", err)
	}

	clog.Debug("密钥生成完成", clog.String("addr", address))
	return address, encryptedData, nil
}

// SignMessage 使用MPC执行消息签名流程。
// 从安全芯片获取密钥，解密数据后进行签名操作。
//
// 参数:
//   - ctx: 上下文，用于控制操作超时或取消
//   - parties: 参与方列表
//   - data: 待签名的消息数据
//   - filename: 临时文件名
//   - userName: 用户名，用于从安全芯片读取数据
//   - address: 以太坊地址
//   - encryptedKey: 加密后的密钥数据
//   - signature: 安全芯片读取操作的授权签名
//
// 返回:
//   - string: 签名结果
//   - error: 操作过程中遇到的错误
func (s *MPCService) SignMessage(ctx context.Context, parties, data, filename, userName, address string, encryptedKey, signature []byte) (string, error) {
	clog.Debug("开始消息签名",
		clog.String("parties", parties),
		clog.String("addr", address),
		clog.String("file", filename))

	// 检查data是否有0x前缀，如果有则移除
	if strings.HasPrefix(data, "0x") {
		data = data[2:]
		clog.Debug("移除数据0x前缀", clog.String("data", data))
	}

	// 确保临时目录存在
	if err := utils.EnsureDir(s.cfg.TempDir); err != nil {
		clog.Error("创建临时目录失败", clog.Err(err))
		return "", fmt.Errorf("创建临时目录失败: %w", err)
	}

	// 构建文件路径
	filePath := filepath.Join(s.cfg.TempDir, filename)
	clog.Debug("临时文件路径", clog.String("path", filePath))

	// 清理函数（在函数返回前删除临时文件）
	defer func() {
		if err := utils.DeleteFile(filePath); err != nil {
			clog.Warn("清理临时文件失败", clog.Err(err))
		}
	}()

	// 从安全芯片读取随机数
	clog.Info("从安全芯片读取密钥")
	randomKey, err := s.securityService.ReadData(userName, address, signature)
	if err != nil {
		clog.Error("读取密钥失败", clog.Err(err))
		return "", fmt.Errorf("从安全芯片读取密钥失败: %w", err)
	}

	// 使用随机数解密数据
	clog.Info("解密密钥数据")
	decryptedData, err := utils.DecryptAES(encryptedKey, randomKey)
	if err != nil {
		clog.Error("解密数据失败", clog.Err(err))
		return "", fmt.Errorf("解密数据失败: %w", err)
	}
	clog.Debug("解密后数据大小", clog.Int("size", len(decryptedData)))

	// 解压数据
	decompressedData, err := utils.DecompressData(decryptedData)
	if err != nil {
		clog.Error("解压数据失败", clog.Err(err))
		return "", fmt.Errorf("解压数据失败: %w", err)
	}
	clog.Debug("解压后数据大小", clog.Int("size", len(decompressedData)))

	// 将解密后的数据写入临时文件
	if err := utils.WriteFile(filePath, decompressedData); err != nil {
		clog.Error("写入临时文件失败", clog.Err(err))
		return "", fmt.Errorf("写入临时文件失败: %w", err)
	}

	// 运行签名命令
	clog.Info("执行MPC签名")
	signResult, err := utils.RunSigning(ctx, s.cfg, parties, data, filePath)
	if err != nil {
		clog.Error("签名失败", clog.Err(err))
		return "", fmt.Errorf("签名失败: %w", err)
	}

	// 将签名转换为以太坊格式
	signResult, _ = utils.ConvertToEthSignature(signResult)

	// 格式化签名结果
	signResult = strings.TrimSpace(signResult)
	if !strings.HasPrefix(signResult, "0x") {
		signResult = "0x" + signResult
	}

	clog.Debug("签名完成", clog.String("sig", signResult))
	return signResult, nil
}

// GetCPLC 获取安全芯片的CPLC信息。
// CPLC(Card Production Life Cycle)包含了芯片制造商、批次、序列号等信息。
//
// 返回:
//   - []byte: CPLC信息
//   - error: 获取过程中遇到的错误
func (s *MPCService) GetCPLC() ([]byte, error) {
	clog.Debug("获取安全芯片CPLC信息")

	cplc, err := s.securityService.GetCPLC()
	if err != nil {
		clog.Error("获取CPLC失败", clog.Err(err))
		return nil, fmt.Errorf("获取安全芯片CPLC信息失败: %w", err)
	}

	clog.Debug("获取CPLC成功", clog.Int("size", len(cplc)))
	return cplc, nil
}
