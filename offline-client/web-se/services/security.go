package services

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"web-se/config"
	"web-se/seclient"
	"web-se/utils"
)

// SecurityService 安全芯片服务
type SecurityService struct {
	cfg        *config.Config
	cardReader *seclient.CardReader
}

// NewSecurityService 创建安全芯片服务
func NewSecurityService(cfg *config.Config) (*SecurityService, error) {
	utils.LogInfo("初始化安全芯片服务")

	// 创建安全芯片服务
	service := &SecurityService{
		cfg: cfg,
	}

	// 创建卡片读取器
	utils.LogDebug("创建卡片读取器")
	reader, err := seclient.NewCardReader(seclient.WithDebug(cfg.Debug))
	if err != nil {
		utils.LogError("创建卡片读取器失败", utils.Error(err))
		return nil, err
	}

	// 连接读卡器
	utils.LogDebug("连接读卡器")
	if err := reader.Connect(""); err != nil {
		utils.LogError("连接读卡器失败", utils.Error(err))
		reader.Close()
		return nil, err
	}

	// 选择Applet
	utils.LogDebug("选择安全芯片Applet")
	if err := reader.SelectApplet(); err != nil {
		utils.LogError("选择Applet失败", utils.Error(err))
		reader.Close()
		return nil, err
	}

	service.cardReader = reader
	utils.LogInfo("安全芯片服务初始化成功")

	return service, nil
}

// Close 关闭安全芯片服务
func (s *SecurityService) Close() {
	if s.cardReader != nil {
		utils.LogInfo("关闭安全芯片服务")
		s.cardReader.Close()
		s.cardReader = nil
	}
}

// StoreData 在安全芯片中存储数据
func (s *SecurityService) StoreData(username, addr string, key []byte) error {
	if s.cardReader == nil {
		utils.LogError("安全芯片服务未启用")
		return errors.New("安全芯片服务未启用")
	}

	utils.LogDebug("准备存储数据到安全芯片",
		utils.String("username", username),
		utils.String("address", addr))

	// 将用户名哈希为 32 字节
	userHash := sha256.Sum256([]byte(username))
	userBytes := userHash[:] // 32 字节

	// 处理以太坊地址，去掉 0x 前缀并转为 20 字节
	cleanAddr := addr
	if len(addr) >= 2 && addr[:2] == "0x" {
		cleanAddr = addr[2:]
	}
	addrBytes, err := hex.DecodeString(cleanAddr)
	if err != nil || len(addrBytes) != 20 {
		utils.LogError("地址格式错误", utils.String("address", addr))
		return errors.New("地址格式错误")
	}

	utils.LogDebug("用户名字节", utils.String("userBytes", fmt.Sprintf("%X", userBytes)))
	utils.LogDebug("地址字节", utils.String("addrBytes", fmt.Sprintf("%X", addrBytes)))

	// 确保密钥长度正确
	if len(key) != seclient.MESSAGE_LENGTH {
		errMsg := fmt.Sprintf("密钥长度必须是 %d 字节", seclient.MESSAGE_LENGTH)
		utils.LogError(errMsg, utils.Int("actual_length", len(key)))
		return errors.New(errMsg)
	}

	// 调用安全芯片存储数据
	utils.LogDebug("调用安全芯片存储数据")
	index, count, err := s.cardReader.StoreData(userBytes, addrBytes, key)
	if err != nil {
		utils.LogError("安全芯片存储数据失败", utils.Error(err))
		return err
	}

	utils.LogInfo("数据已存储到安全芯片",
		utils.Int("index", int(index)),
		utils.Int("total_records", int(count)))
	return nil
}

// ReadData 从安全芯片中读取数据
func (s *SecurityService) ReadData(username, addr string, signature []byte) ([]byte, error) {
	if s.cardReader == nil {
		utils.LogError("安全芯片服务未启用")
		return nil, errors.New("安全芯片服务未启用")
	}

	utils.LogDebug("准备从安全芯片读取数据",
		utils.String("username", username),
		utils.String("address", addr))

	// 将用户名哈希为 32 字节
	userHash := sha256.Sum256([]byte(username))
	userBytes := userHash[:] // 32 字节

	// 处理以太坊地址，去掉 0x 前缀并转为 20 字节
	cleanAddr := addr
	if len(addr) >= 2 && addr[:2] == "0x" {
		cleanAddr = addr[2:]
	}
	addrBytes, err := hex.DecodeString(cleanAddr)
	if err != nil || len(addrBytes) != 20 {
		utils.LogError("地址格式错误", utils.String("address", addr))
		return nil, errors.New("地址格式错误")
	}

	utils.LogDebug("用户名字节", utils.String("userBytes", fmt.Sprintf("%X", userBytes)))
	utils.LogDebug("地址字节", utils.String("addrBytes", fmt.Sprintf("%X", addrBytes)))

	// 调用安全芯片读取数据
	utils.LogDebug("调用安全芯片读取数据",
		utils.String("signature_length", utils.FormatByteSize(int64(len(signature)))))
	data, err := s.cardReader.ReadData(userBytes, addrBytes, signature)
	if err != nil {
		utils.LogError("从安全芯片读取数据失败", utils.Error(err))
		return nil, err
	}

	utils.LogInfo("已从安全芯片读取数据",
		utils.String("data_length", utils.FormatByteSize(int64(len(data)))))
	return data, nil
}

// DeleteData 从安全芯片中删除数据
func (s *SecurityService) DeleteData(username, addr string, signature []byte) error {
	if s.cardReader == nil {
		utils.LogError("安全芯片服务未启用")
		return errors.New("安全芯片服务未启用")
	}

	utils.LogDebug("准备从安全芯片删除数据",
		utils.String("username", username),
		utils.String("address", addr))

	// 将用户名哈希为 32 字节
	userHash := sha256.Sum256([]byte(username))
	userBytes := userHash[:] // 32 字节

	// 处理以太坊地址，去掉 0x 前缀并转为 20 字节
	cleanAddr := addr
	if len(addr) >= 2 && addr[:2] == "0x" {
		cleanAddr = addr[2:]
	}
	addrBytes, err := hex.DecodeString(cleanAddr)
	if err != nil || len(addrBytes) != 20 {
		utils.LogError("地址格式错误", utils.String("address", addr))
		return errors.New("地址格式错误")
	}

	utils.LogDebug("用户名字节", utils.String("userBytes", fmt.Sprintf("%X", userBytes)))
	utils.LogDebug("地址字节", utils.String("addrBytes", fmt.Sprintf("%X", addrBytes)))

	// 调用安全芯片删除数据
	utils.LogDebug("调用安全芯片删除数据")
	index, count, err := s.cardReader.DeleteData(userBytes, addrBytes, signature)
	if err != nil {
		utils.LogError("从安全芯片删除数据失败", utils.Error(err))
		return err
	}

	utils.LogInfo("数据已从安全芯片删除",
		utils.Int("index", int(index)),
		utils.Int("remaining_records", int(count)))
	return nil
}
