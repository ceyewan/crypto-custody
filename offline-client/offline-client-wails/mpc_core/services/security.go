package services

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"offline-client-wails/mpc_core/clog"
	"offline-client-wails/mpc_core/config"
	"offline-client-wails/mpc_core/seclient"
)

// SecurityService 提供与安全芯片通信的服务接口。
// 它处理安全芯片数据的存储、读取和删除操作。
// 该服务被设计为无状态的，每次操作都会建立和关闭连接，以支持硬件热插拔。
type SecurityService struct {
	cfg *config.Config
}

// NewSecurityService 创建并初始化安全芯片服务实例。
// 它只存储配置，不在初始化时建立连接。
func NewSecurityService(cfg *config.Config) (*SecurityService, error) {
	clog.Debug("初始化安全芯片服务", clog.Bool("debug", cfg.Debug))
	service := &SecurityService{cfg: cfg}
	clog.Debug("安全芯片服务初始化完成 (无连接)")
	return service, nil
}

// Close 在无状态模式下，此方法为空，因为没有持久连接。
func (s *SecurityService) Close() {
	clog.Info("关闭安全芯片服务 (无操作)")
}

// connectAndSelect 是一个辅助函数，用于在每次操作前建立连接并选择Applet。
// 它返回一个可用的 CardReader 实例或一个错误。
func (s *SecurityService) connectAndSelect() (*seclient.CardReader, error) {
	// 1. 从配置解析AID
	aid, err := hex.DecodeString(s.cfg.AppletAID)
	if err != nil {
		clog.Error("无法解析配置中的Applet AID", clog.String("aid", s.cfg.AppletAID), clog.Err(err))
		return nil, fmt.Errorf("无效的Applet AID配置: %v", err)
	}

	// 2. 创建读卡器实例
	reader, err := seclient.NewCardReader(seclient.WithDebug(s.cfg.Debug))
	if err != nil {
		clog.Error("创建卡片读取器失败", clog.Err(err))
		return nil, err
	}

	// 3. 连接读卡器
	if err := reader.Connect(s.cfg.CardReaderName); err != nil {
		clog.Error("连接读卡器失败", clog.Err(err))
		reader.Close() // 确保部分成功的资源被释放
		return nil, err
	}

	// 4. 选择Applet
	if err := reader.SelectApplet(aid); err != nil {
		clog.Error("选择Applet失败", clog.Err(err))
		reader.Close() // 确保部分成功的资源被释放
		return nil, err
	}

	return reader, nil
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
	// --- 参数校验 ---
	if username == "" {
		return errors.New("用户名不能为空")
	}
	if addr == "" {
		return errors.New("地址不能为空")
	}
	if len(key) != seclient.MESSAGE_LENGTH {
		errMsg := fmt.Sprintf("密钥长度必须是 %d 字节", seclient.MESSAGE_LENGTH)
		clog.Error(errMsg, clog.Int("actual", len(key)))
		return errors.New(errMsg)
	}
	// --- 参数校验结束 ---

	clog.Debug("存储数据",
		clog.String("username", username),
		clog.String("addr", addr),
		clog.Int("key_len", len(key)))

	reader, err := s.connectAndSelect()
	if err != nil {
		return err
	}
	defer reader.Close()

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

	// 存储数据
	clog.Info("向安全芯片写入数据")
	index, count, err := reader.StoreData(userBytes, addrBytes, key)
	if err != nil {
		clog.Error("存储数据失败", clog.Err(err))
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
	// --- 参数校验 ---
	if username == "" {
		return nil, errors.New("用户名不能为空")
	}
	if addr == "" {
		return nil, errors.New("地址不能为空")
	}
	if len(signature) == 0 {
		return nil, errors.New("签名不能为空")
	}
	// --- 参数校验结束 ---

	clog.Debug("读取数据",
		clog.String("username", username),
		clog.String("addr", addr),
		clog.Int("sig_len", len(signature)))

	reader, err := s.connectAndSelect()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

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
	data, err := reader.ReadData(userBytes, addrBytes, signature)
	if err != nil {
		clog.Error("读取数据失败", clog.Err(err))
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
	// --- 参数校验 ---
	if username == "" {
		return errors.New("用户名不能为空")
	}
	if addr == "" {
		return errors.New("地址不能为空")
	}
	if len(signature) == 0 {
		return errors.New("签名不能为空")
	}
	// --- 参数校验结束 ---

	clog.Debug("删除数据",
		clog.String("username", username),
		clog.String("addr", addr),
		clog.Int("sig_len", len(signature)))

	reader, err := s.connectAndSelect()
	if err != nil {
		return err
	}
	defer reader.Close()

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
	index, count, err := reader.DeleteData(userBytes, addrBytes, signature)
	if err != nil {
		clog.Error("删除数据失败", clog.Err(err))
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

	reader, err := s.connectAndSelect()
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	cplc, err := reader.GetCPLC()
	if err != nil {
		clog.Error("获取CPLC信息失败", clog.Err(err))
		return nil, err
	}

	clog.Debug("CPLC信息获取成功", clog.Int("长度", len(cplc)), clog.String("CPLC", hex.EncodeToString(cplc)))
	return cplc, nil
}
