package seclient

import (
	"fmt"
	"strings"

	"offline-client-wails/mpc_core/clog"

	"github.com/ebfe/scard"
)

// 应用AID (Applet Identifier) - 与 build.xml 中定义的一致
var AID = []byte{0xA0, 0x00, 0x00, 0x00, 0x62, 0xCF, 0x01, 0x01}

// CardReader 结构体封装了与卡片交互的功能
type CardReader struct {
	context  *scard.Context
	card     *scard.Card
	protocol scard.Protocol
	debug    bool   // 调试模式
	cplcData []byte // 缓存CPLC数据
}

// CardReaderOption 是配置 CardReader 的选项函数类型
type CardReaderOption func(*CardReader)

// WithDebug 启用或禁用调试模式
func WithDebug(debug bool) CardReaderOption {
	return func(r *CardReader) {
		r.debug = debug
	}
}

// NewCardReader 初始化读卡器
func NewCardReader(opts ...CardReaderOption) (*CardReader, error) {
	// 建立上下文
	context, err := scard.EstablishContext()
	if err != nil {
		return nil, fmt.Errorf("无法建立智能卡上下文: %v", err)
	}

	reader := &CardReader{
		context: context,
		debug:   false, // 默认不启用调试
	}

	// 应用配置选项
	for _, opt := range opts {
		opt(reader)
	}

	return reader, nil
}

// Close 关闭连接
func (r *CardReader) Close() {
	if r.card != nil {
		r.card.Disconnect(scard.LeaveCard)
		r.card = nil
	}
	if r.context != nil {
		r.context.Release()
		r.context = nil
	}
}

// ListReaders 列出所有可用的读卡器
func (r *CardReader) ListReaders() ([]string, error) {
	return r.context.ListReaders()
}

// Connect 连接到读卡器
func (r *CardReader) Connect(readerName string) error {
	// 获取读卡器列表
	readers, err := r.context.ListReaders()
	if err != nil {
		return fmt.Errorf("无法获取读卡器列表: %v", err)
	}

	if len(readers) == 0 {
		return fmt.Errorf("未找到可用的读卡器")
	}

	// 选择读卡器
	var selectedReader string
	if readerName != "" {
		for _, reader := range readers {
			if strings.Contains(reader, readerName) {
				selectedReader = reader
				break
			}
		}
		if selectedReader == "" {
			if r.debug {
				clog.Info("可用读卡器列表:")
				for i, reader := range readers {
					clog.Infof("  %d: %s", i, reader)
				}
			}
			return fmt.Errorf("未找到包含 '%s' 的读卡器", readerName)
		}
	} else {
		selectedReader = readers[0]
		if r.debug {
			clog.Info("可用读卡器列表:")
			for i, reader := range readers {
				clog.Infof("  %d: %s", i, reader)
				if i == 0 {
					clog.Info("  >>> 自动选择了第一个读卡器")
				}
			}
		}
	}

	if r.debug {
		clog.Infof("使用读卡器: %s", selectedReader)
	}

	// 连接读卡器
	card, err := r.context.Connect(selectedReader, scard.ShareShared, scard.ProtocolAny)
	if err != nil {
		return fmt.Errorf("连接到读卡器失败: %v", err)
	}

	r.card = card
	r.protocol = card.ActiveProtocol()
	if r.debug {
		clog.Infof("成功连接到读卡器，使用协议: %v", r.protocol)
	}

	return nil
}

// SelectApplet 选择Applet
func (r *CardReader) SelectApplet() error {
	// 在选择Applet之前，先获取芯片的CPLC数据，缓存下来
	_, err := r.GetCPLC()
	if err != nil {
		return fmt.Errorf("获取CPLC数据失败: %v", err)
	}
	selectCmd := append([]byte{0x00, 0xA4, 0x04, 0x00, byte(len(AID))}, AID...)

	if r.debug {
		clog.Info("=== 选择Applet命令 ===")
		clog.Infof("APDU: %X", selectCmd)
		clog.Info("命令解析:")
		clog.Info("  CLA: 0x00 (ISO标准命令)")
		clog.Info("  INS: 0xA4 (选择指令)")
		clog.Info("  P1: 0x04 (按名称选择)")
		clog.Info("  P2: 0x00 (首次选择)")
		clog.Infof("  Lc: 0x%02X (AID长度)", len(AID))
		clog.Infof("  Data: %X (AID)", AID)
	}

	resp, err := r.card.Transmit(selectCmd)
	if err != nil {
		return fmt.Errorf("发送选择Applet命令失败: %v", err)
	}

	sw, data := extractResponseAndSW(resp)
	if sw != SW_SUCCESS {
		return fmt.Errorf("选择Applet返回错误状态码: 0x%04X", sw)
	}

	if r.debug {
		clog.Info("=== 选择Applet响应 ===")
		clog.Infof("响应数据: %X", resp)
		clog.Infof("状态码: 0x%04X (成功)", sw)
		clog.Infof("数据: %X", data)
		clog.Infof("成功选择Applet, AID: %X", AID)
	}
	return nil
}

// TransmitAPDU 直接发送APDU命令并返回响应
func (r *CardReader) TransmitAPDU(command []byte) ([]byte, uint16, error) {
	if r.debug {
		clog.Info("=== 发送APDU命令 ===")
		clog.Infof("命令: %X", command)
	}

	resp, err := r.card.Transmit(command)
	if err != nil {
		return nil, 0, fmt.Errorf("发送APDU命令失败: %v", err)
	}

	sw, data := extractResponseAndSW(resp)

	if r.debug {
		clog.Infof("响应状态码: 0x%04X", sw)
		clog.Infof("响应数据: %X", data)
	}

	return data, sw, nil
}
