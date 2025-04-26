package services

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"web-se/clog"

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
	clog.Info("初始化安全芯片服务")

	// 创建安全芯片服务
	service := &SecurityService{
		cfg: cfg,
	}

	// 创建卡片读取器
	clog.Debug("创建卡片读取器")
	reader, err := seclient.NewCardReader(seclient.WithDebug(cfg.Debug))
	if err != nil {
		clog.Error("创建卡片读取器失败", clog.String("error", err.Error()))
		return nil, err
	}

	// 连接读卡器
	clog.Debug("连接读卡器")
	if err := reader.Connect(""); err != nil {
		clog.Error("连接读卡器失败", clog.String("error", err.Error()))
		reader.Close()
		return nil, err
	}

	// 选择Applet
	clog.Debug("选择安全芯片Applet")
	if err := reader.SelectApplet(); err != nil {
		clog.Error("选择Applet失败", clog.String("error", err.Error()))
		reader.Close()
		return nil, err
	}

	service.cardReader = reader
	clog.Info("安全芯片服务初始化成功")

	return service, nil
}

// Close 关闭安全芯片服务
func (s *SecurityService) Close() {
	if s.cardReader != nil {
		clog.Info("关闭安全芯片服务")
		s.cardReader.Close()
		s.cardReader = nil
	}
}

// StoreData 在安全芯片中存储数据
func (s *SecurityService) StoreData(username, addr string, key []byte) error {
	if s.cardReader == nil {
		clog.Error("安全芯片服务未启用")
		return errors.New("安全芯片服务未启用")
	}

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
		clog.Error("地址格式错误", clog.String("address", addr))
		return errors.New("地址格式错误")
	}

	clog.Debug("❕准备存储数据到安全芯片❕")
	clog.Debug("用户名字节", clog.String("userBytes", hex.EncodeToString(userBytes)))
	clog.Debug("地址字节", clog.String("addrBytes", hex.EncodeToString(addrBytes)))
	clog.Debug("密钥字节", clog.String("key", hex.EncodeToString(key)))

	// 确保密钥长度正确
	if len(key) != seclient.MESSAGE_LENGTH {
		errMsg := fmt.Sprintf("密钥长度必须是 %d 字节", seclient.MESSAGE_LENGTH)
		clog.Error(errMsg, clog.Int("actual_length", len(key)))
		return errors.New(errMsg)
	}

	// 调用安全芯片存储数据
	clog.Debug("调用安全芯片存储数据")
	index, count, err := s.cardReader.StoreData(userBytes, addrBytes, key)
	if err != nil {
		clog.Error("安全芯片存储数据失败", clog.String("error", err.Error()))
		return err
	}

	clog.Info("数据已存储到安全芯片",
		clog.Int("index", int(index)),
		clog.Int("total_records", int(count)))
	return nil
}

// ReadData 从安全芯片中读取数据
func (s *SecurityService) ReadData(username, addr string, signature []byte) ([]byte, error) {
	if s.cardReader == nil {
		clog.Error("安全芯片服务未启用")
		return nil, errors.New("安全芯片服务未启用")
	}

	clog.Debug("准备从安全芯片读取数据",
		clog.String("username", username),
		clog.String("address", addr))

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
		clog.Error("地址格式错误", clog.String("address", addr))
		return nil, errors.New("地址格式错误")
	}

	clog.Debug("用户名字节", clog.String("userBytes", fmt.Sprintf("%X", userBytes)))
	clog.Debug("地址字节", clog.String("addrBytes", fmt.Sprintf("%X", addrBytes)))

	// 调用安全芯片读取数据
	clog.Debug("调用安全芯片读取数据",
		clog.String("signature_length", utils.FormatByteSize(int64(len(signature)))))
	data, err := s.cardReader.ReadData(userBytes, addrBytes, signature)
	if err != nil {
		clog.Error("从安全芯片读取数据失败", clog.String("error", err.Error()))
		return nil, err
	}

	clog.Info("已从安全芯片读取数据",
		clog.String("data_length", utils.FormatByteSize(int64(len(data)))))
	return data, nil
}

// DeleteData 从安全芯片中删除数据
func (s *SecurityService) DeleteData(username, addr string, signature []byte) error {
	if s.cardReader == nil {
		clog.Error("安全芯片服务未启用")
		return errors.New("安全芯片服务未启用")
	}

	clog.Debug("准备从安全芯片删除数据",
		clog.String("username", username),
		clog.String("address", addr))

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
		clog.Error("地址格式错误", clog.String("address", addr))
		return errors.New("地址格式错误")
	}

	clog.Debug("用户名字节", clog.String("userBytes", fmt.Sprintf("%X", userBytes)))
	clog.Debug("地址字节", clog.String("addrBytes", fmt.Sprintf("%X", addrBytes)))

	// 调用安全芯片删除数据
	clog.Debug("调用安全芯片删除数据")
	index, count, err := s.cardReader.DeleteData(userBytes, addrBytes, signature)
	if err != nil {
		clog.Error("从安全芯片删除数据失败", clog.String("error", err.Error()))
		return err
	}

	clog.Info("数据已从安全芯片删除",
		clog.Int("index", int(index)),
		clog.Int("remaining_records", int(count)))
	return nil
}
