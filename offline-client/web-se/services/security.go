package services

import (
	"errors"
	"fmt"
	"log"

	"web-se/config"
	"web-se/seclient"
)

// SecurityService 安全芯片服务
type SecurityService struct {
	cfg        *config.Config
	cardReader *seclient.CardReader
}

// NewSecurityService 创建安全芯片服务
func NewSecurityService(cfg *config.Config) (*SecurityService, error) {
	// 创建安全芯片服务
	service := &SecurityService{
		cfg: cfg,
	}

	// 如果配置了使用安全芯片
	if cfg.SecurityChip {
		// 创建卡片读取器
		reader, err := seclient.NewCardReader(seclient.WithDebug(cfg.Debug))
		if err != nil {
			return nil, err
		}

		// 连接读卡器
		if err := reader.Connect(""); err != nil {
			reader.Close()
			return nil, err
		}

		// 选择Applet
		if err := reader.SelectApplet(); err != nil {
			reader.Close()
			return nil, err
		}

		service.cardReader = reader
	}

	return service, nil
}

// Close 关闭安全芯片服务
func (s *SecurityService) Close() {
	if s.cardReader != nil {
		s.cardReader.Close()
		s.cardReader = nil
	}
}

// StoreData 在安全芯片中存储数据
func (s *SecurityService) StoreData(username, addr string, key []byte) error {
	if !s.cfg.SecurityChip || s.cardReader == nil {
		return errors.New("安全芯片服务未启用")
	}

	// 将用户名和地址转为固定长度的字节数组
	userBytes := make([]byte, seclient.USERNAME_LENGTH)
	copy(userBytes, []byte(username))

	addrBytes := make([]byte, seclient.ADDR_LENGTH)
	copy(addrBytes, []byte(addr))

	// 确保密钥长度正确
	if len(key) != seclient.MESSAGE_LENGTH {
		return fmt.Errorf("密钥长度必须是 %d 字节", seclient.MESSAGE_LENGTH)
	}

	// 调用安全芯片存储数据
	index, count, err := s.cardReader.StoreData(userBytes, addrBytes, key)
	if err != nil {
		return err
	}

	log.Printf("数据已存储到安全芯片，索引: %d, 总记录数: %d", index, count)
	return nil
}

// ReadData 从安全芯片中读取数据
func (s *SecurityService) ReadData(username, addr string, signature []byte) ([]byte, error) {
	if !s.cfg.SecurityChip || s.cardReader == nil {
		return nil, errors.New("安全芯片服务未启用")
	}

	// 将用户名和地址转为固定长度的字节数组
	userBytes := make([]byte, seclient.USERNAME_LENGTH)
	copy(userBytes, []byte(username))

	addrBytes := make([]byte, seclient.ADDR_LENGTH)
	copy(addrBytes, []byte(addr))

	// 调用安全芯片读取数据
	data, err := s.cardReader.ReadData(userBytes, addrBytes, signature)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// DeleteData 从安全芯片中删除数据
func (s *SecurityService) DeleteData(username, addr string, signature []byte) error {
	if !s.cfg.SecurityChip || s.cardReader == nil {
		return errors.New("安全芯片服务未启用")
	}

	// 将用户名和地址转为固定长度的字节数组
	userBytes := make([]byte, seclient.USERNAME_LENGTH)
	copy(userBytes, []byte(username))

	addrBytes := make([]byte, seclient.ADDR_LENGTH)
	copy(addrBytes, []byte(addr))

	// 调用安全芯片删除数据
	index, count, err := s.cardReader.DeleteData(userBytes, addrBytes, signature)
	if err != nil {
		return err
	}

	log.Printf("数据已从安全芯片删除，索引: %d, 剩余记录数: %d", index, count)
	return nil
}
