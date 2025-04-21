package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ebfe/scard"
)

// 应用AID (Applet Identifier) - 与 build.xml 中定义的一致
var AID = []byte{0xA0, 0x00, 0x00, 0x00, 0x62, 0xCF, 0x01, 0x01}

// APDU指令常量
const (
	CLA                 = 0x80 // 命令类
	INS_READ_INIT       = 0x20 // 读取数据初始化命令
	INS_READ_CONTINUE   = 0x21 // 读取数据继续命令
	INS_READ_FINALIZE   = 0x22 // 读取数据完成命令
	SW_SUCCESS          = 0x9000
	SW_MORE_DATA_PREFIX = 0x6100
)

// 预定义的凭证列表 - 按优先级排序，程序会按顺序尝试读取
var predefinedCredentials = []Credentials{
	{Username: "User1", Address: "Beijing Haidian"},
	{Username: "User2", Address: "Shanghai Pudong"},
	// 可以添加更多预定义凭证
}

// 查找记录的凭证
type Credentials struct {
	Username string
	Address  string
}

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
	resp, err := r.card.Transmit(selectCmd)
	if err != nil {
		return fmt.Errorf("发送选择Applet命令失败: %v", err)
	}

	sw, err := checkResponse(resp)
	if err != nil {
		return fmt.Errorf("选择Applet失败: %v", err)
	}
	if sw != SW_SUCCESS {
		return fmt.Errorf("选择Applet返回错误状态码: 0x%04X", sw)
	}

	log.Printf("成功选择Applet, AID: %X\n", AID)
	return nil
}

// 读取数据
func (r *CardReader) ReadData(creds Credentials) (string, error) {
	// 1. 读取初始化
	messageLength, err := r.readDataInit(creds)
	if err != nil {
		return "", fmt.Errorf("读取初始化失败: %v", err)
	}
	log.Printf("消息总长度: %d 字节\n", messageLength)

	// 2. 读取消息数据
	message, err := r.readDataContinue(messageLength)
	if err != nil {
		return "", fmt.Errorf("读取数据失败: %v", err)
	}

	// 3. 完成读取操作 (可选)
	// err = r.readDataFinalize()
	// if err != nil {
	//     return "", fmt.Errorf("完成读取操作失败: %v", err)
	// }

	return message, nil
}

// 读取数据初始化
func (r *CardReader) readDataInit(creds Credentials) (uint16, error) {
	// 将用户名和地址转换为UTF-8字节
	usernameBytes := []byte(creds.Username)
	addressBytes := []byte(creds.Address)

	// 检查长度限制
	if len(usernameBytes) > 32 {
		return 0, fmt.Errorf("用户名过长 (最大32字节)")
	}
	if len(addressBytes) > 64 {
		return 0, fmt.Errorf("地址过长 (最大64字节)")
	}

	// 构建APDU命令数据: [userNameLength(1)][userName(var)][addrLength(1)][addr(var)][signatureLength(1)]
	data := []byte{byte(len(usernameBytes))}
	data = append(data, usernameBytes...)
	data = append(data, byte(len(addressBytes)))
	data = append(data, addressBytes...)
	data = append(data, 0) // 签名占位符长度为0

	// 构建完整的APDU命令
	command := []byte{CLA, INS_READ_INIT, 0x00, 0x00, byte(len(data))}
	command = append(command, data...)
	command = append(command, 0x02) // Le字段，期望返回2字节的消息长度

	log.Printf("发送读取初始化命令: %X\n", command)

	// 发送命令
	resp, err := r.card.Transmit(command)
	if err != nil {
		return 0, fmt.Errorf("发送读取初始化命令失败: %v", err)
	}

	sw, err := checkResponse(resp)
	if err != nil {
		return 0, fmt.Errorf("读取初始化失败: %v", err)
	}
	if sw != SW_SUCCESS {
		return 0, fmt.Errorf("读取初始化返回错误状态码: 0x%04X", sw)
	}

	// 解析消息长度（响应中的前2个字节）
	if len(resp) < 2 {
		return 0, fmt.Errorf("响应数据不足，无法获取消息长度")
	}

	// 提取消息长度 (高字节在前)
	messageLength := uint16(resp[0])<<8 | uint16(resp[1])
	return messageLength, nil
}

// 读取数据继续
func (r *CardReader) readDataContinue(messageLength uint16) (string, error) {
	var messageBytes []byte
	moreData := true

	for moreData {
		// 检查是否所有数据都已读取
		if len(messageBytes) >= int(messageLength) {
			moreData = false
			break
		}

		// 计算期望读取的最大长度
		expectedLen := uint8(0xF0) // 一次最多读取240字节
		if uint16(len(messageBytes))+uint16(expectedLen) > messageLength {
			expectedLen = uint8(messageLength - uint16(len(messageBytes)))
		}

		// 构建APDU命令
		command := []byte{CLA, INS_READ_CONTINUE, 0x00, 0x00, expectedLen}

		log.Printf("发送读取继续命令 (期望%d字节): %X\n", expectedLen, command)

		// 发送命令
		resp, err := r.card.Transmit(command)
		if err != nil {
			return "", fmt.Errorf("发送读取继续命令失败: %v", err)
		}

		sw, data := extractResponseAndSW(resp)

		// 添加读取到的数据
		messageBytes = append(messageBytes, data...)

		// 检查状态码
		if sw>>8 == SW_MORE_DATA_PREFIX>>8 { // 还有更多数据 (0x61xx)
			remainingBytes := sw & 0xFF // 剩余字节数 (可能不准确)
			log.Printf("已读取 %d/%d 字节，状态码提示剩余 %d 字节，继续读取...",
				len(messageBytes), messageLength, remainingBytes)
		} else if sw == SW_SUCCESS {
			moreData = false
			log.Printf("已读取 %d/%d 字节，数据接收完毕", len(messageBytes), messageLength)
		} else {
			return "", fmt.Errorf("读取数据返回错误状态码: 0x%04X", sw)
		}
	}

	// 将字节数据转换为字符串
	return string(messageBytes), nil
}

// 读取数据完成 (可选)
func (r *CardReader) readDataFinalize() error {
	command := []byte{CLA, INS_READ_FINALIZE, 0x00, 0x00, 0x00}

	log.Printf("发送读取完成命令: %X\n", command)

	resp, err := r.card.Transmit(command)
	if err != nil {
		return fmt.Errorf("发送读取完成命令失败: %v", err)
	}

	sw, err := checkResponse(resp)
	if err != nil {
		return fmt.Errorf("读取完成失败: %v", err)
	}
	if sw != SW_SUCCESS {
		return fmt.Errorf("读取完成返回错误状态码: 0x%04X", sw)
	}

	return nil
}

// 检查响应，提取状态码
func checkResponse(resp []byte) (uint16, error) {
	if len(resp) < 2 {
		return 0, fmt.Errorf("响应数据过短")
	}

	sw := binary.BigEndian.Uint16(resp[len(resp)-2:])
	return sw, nil
}

// 从响应中提取数据部分和状态码
func extractResponseAndSW(resp []byte) (uint16, []byte) {
	if len(resp) < 2 {
		return 0, []byte{}
	}

	data := resp[:len(resp)-2]
	sw := binary.BigEndian.Uint16(resp[len(resp)-2:])
	return sw, data
}

// 格式化输出十六进制数据
func dumpHex(data []byte) string {
	var buf bytes.Buffer

	for i, b := range data {
		if i > 0 && i%16 == 0 {
			buf.WriteString("\n")
		} else if i > 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(fmt.Sprintf("%02X", b))
	}

	return buf.String()
}

func main() {
	// 设置基本日志格式
	log.SetOutput(os.Stdout)
	log.SetFlags(0)

	fmt.Println("安全芯片读取客户端 - 数据读取工具")
	fmt.Println("======================================")

	// 初始化读卡器
	reader, err := NewCardReader()
	if err != nil {
		log.Fatalf("初始化读卡器失败: %v", err)
	}
	defer reader.Close()

	// 连接到读卡器 (自动选择第一个可用的读卡器)
	err = reader.Connect("")
	if err != nil {
		log.Fatalf("连接到读卡器失败: %v", err)
	}

	// 选择Applet
	err = reader.SelectApplet()
	if err != nil {
		log.Fatalf("选择Applet失败: %v", err)
	}

	fmt.Println("\n开始读取数据，将尝试预定义的凭证列表...")

	// 尝试预定义凭证列表
	var message string

	for i, creds := range predefinedCredentials {
		fmt.Printf("\n[%d] 尝试凭证: 用户名='%s', 地址='%s'\n", i+1, creds.Username, creds.Address)

		msg, err := reader.ReadData(creds)
		if err != nil {
			fmt.Printf("    读取失败: %v\n", err)
			continue
		}

		// 成功读取
		message = msg
		fmt.Printf("    ✓ 凭证有效! 成功读取数据\n")
		fmt.Printf("    消息内容: %s\n", message)
	}
}
