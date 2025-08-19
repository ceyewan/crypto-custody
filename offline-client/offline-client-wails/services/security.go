package services

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"offline-client-wails/clog"
	"offline-client-wails/config"
	"offline-client-wails/seclient"
)

// SecurityService 提供与安全芯片通信的服务接口。
// 它处理安全芯片数据的存储、读取和删除操作。
type SecurityService struct {
	cfg        *config.Config
	cardReader *seclient.CardReader
}

// NewSecurityService 创建并初始化安全芯片服务实例。
// 它会连接读卡器并选择正确的Applet以进行后续操作。
// 参数:
//   - cfg: 系统配置对象
//
// 返回:
//   - *SecurityService: 初始化后的服务实例
//   - error: 初始化过程中遇到的错误
func NewSecurityService(cfg *config.Config) (*SecurityService, error) {
	clog.Debug("初始化安全芯片服务", clog.Bool("debug", cfg.Debug))

	service := &SecurityService{cfg: cfg}

	// 创建卡片读取器
	clog.Info("创建卡片读取器")
	reader, err := seclient.NewCardReader(seclient.WithDebug(cfg.Debug))
	if err != nil {
		clog.Error("创建卡片读取器失败", clog.String("err", err.Error()))
		return nil, err
	}

	// 连接读卡器
	clog.Info("连接读卡器")
	if err := reader.Connect(cfg.CardReaderName); err != nil {
		clog.Error("连接读卡器失败", clog.String("err", err.Error()))
		reader.Close()
		return nil, err
	}

	// 选择Applet
	clog.Info("选择安全芯片Applet")
	if err := reader.SelectApplet(); err != nil {
		clog.Error("选择Applet失败", clog.String("err", err.Error()))
		reader.Close()
		return nil, err
	}

	service.cardReader = reader
	clog.Debug("安全芯片服务初始化完成")

	return service, nil
}

// Close 关闭安全芯片服务，释放相关资源。
// 此方法应该在服务不再使用时调用。
func (s *SecurityService) Close() {
	if s.cardReader != nil {
		clog.Info("关闭安全芯片服务")
		s.cardReader.Close()
		s.cardReader = nil
	}
}

// StoreData 在安全芯片中存储用户数据。
// 参数:
//   - username: 用户名，将被哈希处理
//   - addr: 以太坊地址，格式可带或不带0x前缀
//   - key: 要存储的数据，长度必须符合seclient.MESSAGE_LENGTH规范
//
// 返回:
//   - error: 存储过程中遇到的错误
func (s *SecurityService) StoreData(username, addr string, key []byte) error {
	clog.Debug("存储数据",
		clog.String("username", username),
		clog.String("addr", addr),
		clog.Int("key_len", len(key)))

	if s.cardReader == nil {
		return errors.New("安全芯片服务未启用")
	}

	// 将用户名哈希为32字节
	userHash := sha256.Sum256([]byte(username))
	userBytes := userHash[:]

	// 处理以太坊地址
	cleanAddr := addr
	if len(addr) >= 2 && addr[:2] == "0x" {
		cleanAddr = addr[2:]
	}

	addrBytes, err := hex.DecodeString(cleanAddr)
	if err != nil || len(addrBytes) != 20 {
		clog.Error("地址格式错误", clog.String("addr", addr))
		return errors.New("地址格式错误")
	}

	// 确保密钥长度正确
	if len(key) != seclient.MESSAGE_LENGTH {
		errMsg := fmt.Sprintf("密钥长度必须是 %d 字节", seclient.MESSAGE_LENGTH)
		clog.Error(errMsg, clog.Int("actual", len(key)))
		return errors.New(errMsg)
	}

	// 存储数据
	clog.Info("向安全芯片写入数据")
	index, count, err := s.cardReader.StoreData(userBytes, addrBytes, key)
	if err != nil {
		clog.Error("存储数据失败", clog.String("err", err.Error()))
		return err
	}

	clog.Debug("数据存储成功", clog.Int("记录索引", int(index)), clog.Int("记录总数", int(count)))
	return nil
}

// ReadData 从安全芯片中读取用户数据。
// 参数:
//   - username: 用户名，将被哈希处理
//   - addr: 以太坊地址，格式可带或不带0x前缀
//   - signature: 读取操作的授权签名
//
// 返回:
//   - []byte: 读取的数据内容
//   - error: 读取过程中遇到的错误
func (s *SecurityService) ReadData(username, addr string, signature []byte) ([]byte, error) {
	clog.Debug("读取数据",
		clog.String("username", username),
		clog.String("addr", addr),
		clog.Int("sig_len", len(signature)))

	if s.cardReader == nil {
		return nil, errors.New("安全芯片服务未启用")
	}

	// 将用户名哈希为32字节
	userHash := sha256.Sum256([]byte(username))
	userBytes := userHash[:]

	// 处理以太坊地址
	cleanAddr := addr
	if len(addr) >= 2 && addr[:2] == "0x" {
		cleanAddr = addr[2:]
	}

	addrBytes, err := hex.DecodeString(cleanAddr)
	if err != nil || len(addrBytes) != 20 {
		clog.Error("地址格式错误", clog.String("addr", addr))
		return nil, errors.New("地址格式错误")
	}

	// 读取数据
	clog.Info("从安全芯片读取数据")
	data, err := s.cardReader.ReadData(userBytes, addrBytes, signature)
	if err != nil {
		clog.Error("读取数据失败", clog.String("err", err.Error()))
		return nil, err
	}

	clog.Debug("数据读取成功", clog.Int("数据长度", len(data)))
	return data, nil
}

// DeleteData 从安全芯片中删除用户数据。
// 参数:
//   - username: 用户名，将被哈希处理
//   - addr: 以太坊地址，格式可带或不带0x前缀
//   - signature: 删除操作的授权签名
//
// 返回:
//   - error: 删除过程中遇到的错误
func (s *SecurityService) DeleteData(username, addr string, signature []byte) error {
	clog.Debug("删除数据",
		clog.String("username", username),
		clog.String("addr", addr),
		clog.Int("sig_len", len(signature)))

	if s.cardReader == nil {
		return errors.New("安全芯片服务未启用")
	}

	// 将用户名哈希为32字节
	userHash := sha256.Sum256([]byte(username))
	userBytes := userHash[:]

	// 处理以太坊地址
	cleanAddr := addr
	if len(addr) >= 2 && addr[:2] == "0x" {
		cleanAddr = addr[2:]
	}

	addrBytes, err := hex.DecodeString(cleanAddr)
	if err != nil || len(addrBytes) != 20 {
		clog.Error("地址格式错误", clog.String("addr", addr))
		return errors.New("地址格式错误")
	}

	// 删除数据
	clog.Info("从安全芯片删除数据")
	index, count, err := s.cardReader.DeleteData(userBytes, addrBytes, signature)
	if err != nil {
		clog.Error("删除数据失败", clog.String("err", err.Error()))
		return err
	}

	clog.Debug("数据删除成功", clog.Int("记录索引", int(index)), clog.Int("记录总数", int(count)))
	return nil
}

// GetCPLC 获取安全芯片的CPLC信息
// 返回:
//   - []byte: CPLC信息
//   - error: 获取过程中遇到的错误
func (s *SecurityService) GetCPLC() ([]byte, error) {
	clog.Debug("获取CPLC信息")

	if s.cardReader == nil {
		return nil, errors.New("安全芯片服务未启用")
	}

	cplc, err := s.cardReader.GetCPLC()
	if err != nil {
		clog.Error("获取CPLC信息失败", clog.String("err", err.Error()))
		return nil, err
	}

	clog.Debug("CPLC信息获取成功", clog.Int("长度", len(cplc)), clog.String("CPLC", hex.EncodeToString(cplc)))
	return cplc, nil
}
