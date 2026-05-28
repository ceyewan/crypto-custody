package services

import (
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"offline-client-wails/mpc_core/clog"
	"offline-client-wails/mpc_core/config"
	"offline-client-wails/mpc_core/seclient"
)

// SecurityService 提供与安全芯片通信的服务接口。
type SecurityService struct {
	cfg *config.Config
}

// NewSecurityService 创建并初始化安全芯片服务实例。
func NewSecurityService(cfg *config.Config) (*SecurityService, error) {
	clog.Debug("初始化安全芯片服务", clog.Bool("debug", cfg.Debug))
	return &SecurityService{cfg: cfg}, nil
}

// Close 在无状态模式下为空。
func (s *SecurityService) Close() {
	clog.Info("关闭安全芯片服务 (无操作)")
}

func (s *SecurityService) connectAndSelect() (*seclient.CardReader, error) {
	aid, err := hex.DecodeString(s.cfg.AppletAID)
	if err != nil {
		return nil, fmt.Errorf("无效的Applet AID配置: %v", err)
	}

	reader, err := seclient.NewCardReader(seclient.WithDebug(s.cfg.Debug))
	if err != nil {
		return nil, err
	}
	if err := reader.Connect(s.cfg.CardReaderName); err != nil {
		reader.Close()
		return nil, err
	}
	if err := reader.SelectApplet(aid); err != nil {
		reader.Close()
		return nil, err
	}
	return reader, nil
}

// StoreData 在安全芯片中存储 record_id 对应的 32 字节 AES key。
func (s *SecurityService) StoreData(recordID, addr string, key []byte) error {
	if recordID == "" {
		return errors.New("record_id不能为空")
	}
	if addr == "" {
		return errors.New("地址不能为空")
	}
	if len(key) != seclient.MESSAGE_LENGTH {
		return fmt.Errorf("密钥长度必须是 %d 字节", seclient.MESSAGE_LENGTH)
	}

	recordBytes, err := parseRecordID(recordID)
	if err != nil {
		return err
	}
	addrBytes, err := parseAddress(addr)
	if err != nil {
		return err
	}

	reader, err := s.connectAndSelect()
	if err != nil {
		return err
	}
	defer reader.Close()

	index, count, err := reader.StoreData(recordBytes, addrBytes, key)
	if err != nil {
		return err
	}
	clog.Debug("SE数据存储成功", clog.Int("记录索引", int(index)), clog.Int("记录总数", int(count)))
	return nil
}

// ReadData 从安全芯片中读取 record_id 对应数据。
func (s *SecurityService) ReadData(recordID, addr string, signature []byte) ([]byte, error) {
	if recordID == "" {
		return nil, errors.New("record_id不能为空")
	}
	if addr == "" {
		return nil, errors.New("地址不能为空")
	}
	if len(signature) == 0 {
		return nil, errors.New("签名不能为空")
	}

	recordBytes, err := parseRecordID(recordID)
	if err != nil {
		return nil, err
	}
	addrBytes, err := parseAddress(addr)
	if err != nil {
		return nil, err
	}

	reader, err := s.connectAndSelect()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	data, err := reader.ReadData(recordBytes, addrBytes, signature)
	if err != nil {
		return nil, err
	}
	clog.Debug("SE数据读取成功", clog.Int("数据长度", len(data)))
	return data, nil
}

// DeleteData 从安全芯片中删除 record_id 对应数据。
func (s *SecurityService) DeleteData(recordID, addr string, signature []byte) error {
	if recordID == "" {
		return errors.New("record_id不能为空")
	}
	if addr == "" {
		return errors.New("地址不能为空")
	}
	if len(signature) == 0 {
		return errors.New("签名不能为空")
	}

	recordBytes, err := parseRecordID(recordID)
	if err != nil {
		return err
	}
	addrBytes, err := parseAddress(addr)
	if err != nil {
		return err
	}

	reader, err := s.connectAndSelect()
	if err != nil {
		return err
	}
	defer reader.Close()

	index, count, err := reader.DeleteData(recordBytes, addrBytes, signature)
	if err != nil {
		return err
	}
	clog.Debug("SE数据删除成功", clog.Int("记录索引", int(index)), clog.Int("记录总数", int(count)))
	return nil
}

// GetCPLC 获取安全芯片的CPLC信息。
func (s *SecurityService) GetCPLC() ([]byte, error) {
	reader, err := s.connectAndSelect()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	cplc, err := reader.GetCPLC()
	if err != nil {
		return nil, err
	}
	clog.Debug("CPLC信息获取成功", clog.Int("长度", len(cplc)))
	return cplc, nil
}

func parseRecordID(recordID string) ([]byte, error) {
	recordBytes, err := hex.DecodeString(strings.TrimPrefix(recordID, "0x"))
	if err != nil || len(recordBytes) != seclient.USERNAME_LENGTH {
		return nil, fmt.Errorf("record_id必须是%d字节hex", seclient.USERNAME_LENGTH)
	}
	return recordBytes, nil
}

func parseAddress(addr string) ([]byte, error) {
	addrBytes, err := hex.DecodeString(strings.TrimPrefix(addr, "0x"))
	if err != nil || len(addrBytes) != seclient.ADDR_LENGTH {
		return nil, errors.New("地址格式错误")
	}
	return addrBytes, nil
}
