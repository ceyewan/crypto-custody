package seclient

import (
	"fmt"
	"log"
	"strings"

	"github.com/ebfe/scard"
)

// 应用AID (Applet Identifier) - 与 build.xml 中定义的一致
var AID = []byte{0xA0, 0x00, 0x00, 0x00, 0x62, 0xCF, 0x01, 0x01}

// CardReader 结构体封装了与卡片交互的功能
type CardReader struct {
	context  *scard.Context
	card     *scard.Card
	protocol scard.Protocol
}

// 初始化读卡器
func NewCardReader() (*CardReader, error) {
	// 建立上下文
	context, err := scard.EstablishContext()
	if err != nil {
		return nil, fmt.Errorf("无法建立智能卡上下文: %v", err)
	}

	return &CardReader{
		context: context,
	}, nil
}

// 关闭连接
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

// 连接到读卡器
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
			fmt.Println("可用读卡器列表:")
			for i, reader := range readers {
				fmt.Printf("  %d: %s\n", i, reader)
			}
			return fmt.Errorf("未找到包含 '%s' 的读卡器", readerName)
		}
	} else {
		selectedReader = readers[0]
		fmt.Println("可用读卡器列表:")
		for i, reader := range readers {
			fmt.Printf("  %d: %s\n", i, reader)
			if i == 0 {
				fmt.Printf("  >>> 自动选择了第一个读卡器\n")
			}
		}
	}

	log.Printf("使用读卡器: %s\n", selectedReader)

	// 连接读卡器
	card, err := r.context.Connect(selectedReader, scard.ShareShared, scard.ProtocolAny)
	if err != nil {
		return fmt.Errorf("连接到读卡器失败: %v", err)
	}

	r.card = card
	r.protocol = card.ActiveProtocol()
	log.Printf("成功连接到读卡器，使用协议: %v\n", r.protocol)

	return nil
}

// 选择Applet
func (r *CardReader) SelectApplet() error {
	selectCmd := append([]byte{0x00, 0xA4, 0x04, 0x00, byte(len(AID))}, AID...)

	log.Printf("\n=== 选择Applet命令 ===\n")
	log.Printf("APDU: %X\n", selectCmd)
	log.Printf("命令解析:\n")
	log.Printf("  CLA: 0x00 (ISO标准命令)\n")
	log.Printf("  INS: 0xA4 (选择指令)\n")
	log.Printf("  P1: 0x04 (按名称选择)\n")
	log.Printf("  P2: 0x00 (首次选择)\n")
	log.Printf("  Lc: 0x%02X (AID长度)\n", len(AID))
	log.Printf("  Data: %X (AID)\n", AID)

	resp, err := r.card.Transmit(selectCmd)
	if err != nil {
		return fmt.Errorf("发送选择Applet命令失败: %v", err)
	}

	sw, data := extractResponseAndSW(resp)
	if sw != SW_SUCCESS {
		return fmt.Errorf("选择Applet返回错误状态码: 0x%04X", sw)
	}

	log.Printf("\n=== 选择Applet响应 ===\n")
	log.Printf("响应数据: %X\n", resp)
	log.Printf("状态码: 0x%04X (成功)\n", sw)
	log.Printf("数据: %X\n", data)
	log.Printf("成功选择Applet, AID: %X\n", AID)
	return nil
}
