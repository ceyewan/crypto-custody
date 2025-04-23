package main

import (
	crypto_rand "crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/ebfe/scard"
)

// 应用AID (Applet Identifier) - 与 build.xml 中定义的一致
var AID = []byte{0xA0, 0x00, 0x00, 0x00, 0x62, 0xCF, 0x01, 0x01}

// APDU指令常量
const (
	CLA                 = 0x80 // 命令类
	INS_STORE_DATA      = 0x10 // 存储数据命令
	INS_READ_DATA       = 0x20 // 读取数据命令
	INS_DELETE_DATA     = 0x30 // 删除数据命令
	SW_SUCCESS          = 0x9000
	SW_RECORD_NOT_FOUND = 0x6A83
	SW_FILE_FULL        = 0x6A84
	SW_WRONG_LENGTH     = 0x6700

	// 固定长度常量
	USERNAME_LENGTH = 32
	ADDR_LENGTH     = 64
	MESSAGE_LENGTH  = 32
)

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

// 将用户名转换为固定长度的字节数组
func usernameToBytes(username string) []byte {
	// 使用SHA256哈希算法将任意长度的用户名映射为固定32字节
	hash := sha256.Sum256([]byte(username))
	return hash[:]
}

// 确保地址为64字节
func ensureAddrLength(addr []byte) []byte {
	result := make([]byte, ADDR_LENGTH)
	copy(result, addr) // 复制数据，不足的部分将保持为零值(0)
	return result
}

// 确保消息为32字节
func ensureMessageLength(message []byte) []byte {
	result := make([]byte, MESSAGE_LENGTH)
	copy(result, message) // 复制数据，不足的部分将保持为零值(0)
	return result
}

// StoreData 存储数据 - 符合新的APDU格式
func (r *CardReader) StoreData(username string, addr []byte, message []byte) error {
	// 确保输入数据符合长度要求
	usernameBytes := usernameToBytes(username)
	addrBytes := ensureAddrLength(addr)
	messageBytes := ensureMessageLength(message)

	// 构造完整数据
	fullData := make([]byte, 0, USERNAME_LENGTH+ADDR_LENGTH+MESSAGE_LENGTH)
	fullData = append(fullData, usernameBytes...)
	fullData = append(fullData, addrBytes...)
	fullData = append(fullData, messageBytes...)

	// 构建APDU命令
	command := []byte{CLA, INS_STORE_DATA, 0x00, 0x00, byte(len(fullData))}
	command = append(command, fullData...)

	log.Printf("\n=== 存储数据命令 ===\n")
	log.Printf("APDU: %X...(显示前20字节)\n", command[:min(20, len(command))])
	log.Printf("命令解析:\n")
	log.Printf("  CLA: 0x%02X (命令类)\n", CLA)
	log.Printf("  INS: 0x%02X (存储数据指令)\n", INS_STORE_DATA)
	log.Printf("  P1: 0x00\n")
	log.Printf("  P2: 0x00\n")
	log.Printf("  Lc: 0x%02X (数据总长=%d)\n", len(fullData), len(fullData))
	log.Printf("  Data: [用户名(32字节)][地址(64字节)][消息(32字节)]\n")
	log.Printf("    用户名(哈希值): %X\n", usernameBytes[:min(16, len(usernameBytes))])
	log.Printf("    地址(前16字节): %X...\n", addrBytes[:min(16, len(addrBytes))])
	log.Printf("    消息(前16字节): %X...\n", messageBytes[:min(16, len(messageBytes))])

	// 发送命令
	resp, err := r.card.Transmit(command)
	if err != nil {
		return fmt.Errorf("发送存储数据命令失败: %v", err)
	}

	sw, data := extractResponseAndSW(resp)
	if sw != SW_SUCCESS {
		if sw == SW_FILE_FULL {
			return fmt.Errorf("存储失败: 存储空间已满 (状态码: 0x%04X)", sw)
		} else if sw == SW_WRONG_LENGTH {
			return fmt.Errorf("存储失败: 数据长度错误 (状态码: 0x%04X)", sw)
		}
		return fmt.Errorf("存储失败: 未知错误 (状态码: 0x%04X)", sw)
	}

	log.Printf("\n=== 存储数据响应 ===\n")
	log.Printf("响应数据: %X\n", resp)
	log.Printf("状态码: 0x%04X (成功)\n", sw)

	// 解析响应数据
	if len(data) >= 2 {
		recordIndex := data[0]
		recordCount := data[1]
		log.Printf("响应解析:\n")
		log.Printf("  记录索引: 0x%02X (%d)\n", recordIndex, recordIndex)
		log.Printf("  当前记录总数: 0x%02X (%d)\n", recordCount, recordCount)
	}

	return nil
}

// ReadData 读取数据 - 符合新的APDU格式
func (r *CardReader) ReadData(username string, addr []byte) ([]byte, error) {
	// 确保输入数据符合长度要求
	usernameBytes := usernameToBytes(username)
	addrBytes := ensureAddrLength(addr)

	// 构造完整数据
	fullData := make([]byte, 0, USERNAME_LENGTH+ADDR_LENGTH)
	fullData = append(fullData, usernameBytes...)
	fullData = append(fullData, addrBytes...)

	// 构建APDU命令
	command := []byte{CLA, INS_READ_DATA, 0x00, 0x00, byte(len(fullData))}
	command = append(command, fullData...)

	log.Printf("\n=== 读取数据命令 ===\n")
	log.Printf("APDU: %X...(显示前20字节)\n", command[:min(20, len(command))])
	log.Printf("命令解析:\n")
	log.Printf("  CLA: 0x%02X (命令类)\n", CLA)
	log.Printf("  INS: 0x%02X (读取数据指令)\n", INS_READ_DATA)
	log.Printf("  P1: 0x00\n")
	log.Printf("  P2: 0x00\n")
	log.Printf("  Lc: 0x%02X (数据总长=%d)\n", len(fullData), len(fullData))
	log.Printf("  Data: [用户名(32字节)][地址(64字节)]\n")
	log.Printf("    用户名(哈希值): %X\n", usernameBytes[:min(16, len(usernameBytes))])
	log.Printf("    地址(前16字节): %X...\n", addrBytes[:min(16, len(addrBytes))])

	// 发送命令
	resp, err := r.card.Transmit(command)
	if err != nil {
		return nil, fmt.Errorf("发送读取数据命令失败: %v", err)
	}

	sw, data := extractResponseAndSW(resp)
	if sw == SW_RECORD_NOT_FOUND {
		return nil, fmt.Errorf("记录未找到 (状态码: 0x%04X)", sw)
	} else if sw != SW_SUCCESS {
		return nil, fmt.Errorf("读取数据返回错误状态码: 0x%04X", sw)
	}

	log.Printf("\n=== 读取数据响应 ===\n")
	log.Printf("响应数据: %X\n", resp)
	log.Printf("状态码: 0x%04X (成功)\n", sw)
	log.Printf("解析响应: 读取到消息数据 %d 字节\n", len(data))
	log.Printf("消息数据: %X\n", data)

	return data, nil
}

// DeleteData 删除数据 - 根据用户名和地址删除记录
func (r *CardReader) DeleteData(username string, addr []byte) error {
	// 确保输入数据符合长度要求
	usernameBytes := usernameToBytes(username)
	addrBytes := ensureAddrLength(addr)

	// 构造完整数据
	fullData := make([]byte, 0, USERNAME_LENGTH+ADDR_LENGTH)
	fullData = append(fullData, usernameBytes...)
	fullData = append(fullData, addrBytes...)

	// 构建APDU命令
	command := []byte{CLA, INS_DELETE_DATA, 0x00, 0x00, byte(len(fullData))}
	command = append(command, fullData...)

	log.Printf("\n=== 删除数据命令 ===\n")
	log.Printf("APDU: %X...(显示前20字节)\n", command[:min(20, len(command))])
	log.Printf("命令解析:\n")
	log.Printf("  CLA: 0x%02X (命令类)\n", CLA)
	log.Printf("  INS: 0x%02X (删除数据指令)\n", INS_DELETE_DATA)
	log.Printf("  P1: 0x00\n")
	log.Printf("  P2: 0x00\n")
	log.Printf("  Lc: 0x%02X (数据总长=%d)\n", len(fullData), len(fullData))
	log.Printf("  Data: [用户名(32字节)][地址(64字节)]\n")
	log.Printf("    用户名(哈希值): %X\n", usernameBytes[:min(16, len(usernameBytes))])
	log.Printf("    地址(前16字节): %X...\n", addrBytes[:min(16, len(addrBytes))])

	// 发送命令
	resp, err := r.card.Transmit(command)
	if err != nil {
		return fmt.Errorf("发送删除数据命令失败: %v", err)
	}

	sw, data := extractResponseAndSW(resp)
	if sw == SW_RECORD_NOT_FOUND {
		return fmt.Errorf("记录未找到 (状态码: 0x%04X)", sw)
	} else if sw != SW_SUCCESS {
		return fmt.Errorf("删除数据返回错误状态码: 0x%04X", sw)
	}

	log.Printf("\n=== 删除数据响应 ===\n")
	log.Printf("响应数据: %X\n", resp)
	log.Printf("状态码: 0x%04X (成功)\n", sw)
	log.Printf("解析响应: 删除的记录索引=%d, 剩余记录数=%d\n", data[0], data[1])

	return nil
}

// 辅助函数: 从APDU响应中提取状态码和数据
func extractResponseAndSW(resp []byte) (uint16, []byte) {
	if len(resp) < 2 {
		return 0, nil
	}
	sw := (uint16(resp[len(resp)-2]) << 8) | uint16(resp[len(resp)-1])
	data := resp[:len(resp)-2]
	return sw, data
}

// 辅助函数: 取最小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 生成随机消息
func generateRandomMessage(length int) []byte {
	message := make([]byte, length)
	crypto_rand.Read(message)
	return message
}

// 生成随机地址
func generateRandomAddress(length int) []byte {
	addr := make([]byte, length)
	crypto_rand.Read(addr)
	return addr
}

// 基础测试数据
var testData = []struct {
	username string
	addr     []byte
	message  []byte
	desc     string
}{
	{
		username: "user1@example.com",
		addr:     []byte("Beijing Haidian District, No.10 Software Park"),
		message:  []byte("Test message for user1"),
		desc:     "基本消息测试",
	},
	{
		username: "user1@example.com", // 重复用户名/地址以测试覆盖
		addr:     []byte("Beijing Haidian District, No.10 Software Park"),
		message:  []byte("Updated message for user1"),
		desc:     "相同用户名和地址的覆盖测试",
	},
	{
		username: "user2@example.com",
		addr:     []byte("Shanghai Pudong New Area, Century Avenue 100"),
		message:  []byte("Test message for user2"),
		desc:     "不同用户名和地址测试",
	},
	{
		username: "user3@example.com",
		addr:     []byte("Guangzhou Tianhe District, Pearl River New City"),
		message:  []byte{0x01, 0x02, 0x03, 0x04}, // 简短二进制消息
		desc:     "二进制消息测试",
	},
	{
		username: "userWithVeryLongName@verylongdomain.com",
		addr:     []byte("Address with exactly 64 bytes of data padded to fill requirement space"),
		message:  []byte("Message with exactly 32 bytes of data."),
		desc:     "边界长度测试",
	},
}

func runBasicTests(reader *CardReader) {
	fmt.Println("\n=== 基本功能测试 ===")
	log.Printf("\n\n=== 基本功能测试 ===\n")

	// 测试存储和读取
	for i, data := range testData {
		fmt.Printf("\n>> 测试 #%d: %s\n", i+1, data.desc)
		log.Printf("\n>> 测试 #%d: %s\n", i+1, data.desc)

		// 存储数据
		fmt.Printf("   存储 - 用户名: %s\n", data.username)
		log.Printf("   存储 - 用户名: %s\n", data.username)
		log.Printf("   存储 - 地址: %s\n", string(data.addr))
		log.Printf("   存储 - 消息: %s\n", string(data.message))

		err := reader.StoreData(data.username, data.addr, data.message)
		if err != nil {
			fmt.Printf("   存储失败: %v\n", err)
			log.Printf("   存储失败: %v\n", err)
			continue
		}
		fmt.Printf("   存储成功\n")

		// 读取数据进行验证
		fmt.Printf("   读取中...\n")
		retrievedMessage, err := reader.ReadData(data.username, data.addr)
		if err != nil {
			fmt.Printf("   读取失败: %v\n", err)
			log.Printf("   读取失败: %v\n", err)
			continue
		}

		// 比较原始消息和读取的消息
		// 注意：读取的消息总是32字节，需要截断比较
		originalTruncated := ensureMessageLength(data.message)
		matched := compareByteArrays(originalTruncated, retrievedMessage)

		fmt.Printf("   读取成功: %d字节\n", len(retrievedMessage))
		fmt.Printf("   消息验证: %s\n", boolToCheckmark(matched))
		log.Printf("   读取成功: %d字节\n", len(retrievedMessage))
		log.Printf("   原始消息(Hex): %X\n", originalTruncated)
		log.Printf("   读取消息(Hex): %X\n", retrievedMessage)
		log.Printf("   消息验证: %s\n", boolToVerifiedStr(matched))

		if !matched {
			fmt.Printf("   错误: 消息内容不匹配!\n")
			log.Printf("   错误: 消息内容不匹配!\n")
			// 找出第一个不同的字节
			for i := 0; i < min(len(originalTruncated), len(retrievedMessage)); i++ {
				if originalTruncated[i] != retrievedMessage[i] {
					log.Printf("   首个不匹配位置: 索引=%d, 原始=%X, 读取=%X\n",
						i, originalTruncated[i], retrievedMessage[i])
					break
				}
			}
		}
	}
}

func runOverwriteTest(reader *CardReader) {
	fmt.Println("\n=== 覆盖测试 ===")
	log.Printf("\n\n=== 覆盖测试 ===\n")

	username := "overwrite_test_user"
	addr := []byte("Overwrite Test Address - This address should be exactly 64 bytes for testing!")
	originalMessage := []byte("Original message for overwrite test")
	updatedMessage := []byte("Updated message after overwrite test")

	// 存储原始消息
	fmt.Printf(">> 存储原始消息\n")
	log.Printf(">> 存储原始消息\n")
	log.Printf("   用户名: %s\n", username)
	log.Printf("   地址: %s\n", string(addr))
	log.Printf("   原始消息: %s\n", string(originalMessage))

	err := reader.StoreData(username, addr, originalMessage)
	if err != nil {
		fmt.Printf("   存储原始消息失败: %v\n", err)
		log.Printf("   存储原始消息失败: %v\n", err)
		return
	}
	fmt.Printf("   原始消息存储成功\n")

	// 读取确认
	fmt.Printf(">> 读取原始消息以确认\n")
	_, err = reader.ReadData(username, addr)
	if err != nil {
		fmt.Printf("   读取原始消息失败: %v\n", err)
		log.Printf("   读取原始消息失败: %v\n", err)
		return
	}

	// 存储更新消息 (覆盖)
	fmt.Printf(">> 存储更新消息 (覆盖原始消息)\n")
	log.Printf(">> 存储更新消息 (覆盖原始消息)\n")
	log.Printf("   用户名: %s\n", username)
	log.Printf("   地址: %s\n", string(addr))
	log.Printf("   更新消息: %s\n", string(updatedMessage))

	err = reader.StoreData(username, addr, updatedMessage)
	if err != nil {
		fmt.Printf("   存储更新消息失败: %v\n", err)
		log.Printf("   存储更新消息失败: %v\n", err)
		return
	}
	fmt.Printf("   更新消息存储成功\n")

	// 读取更新后的消息确认
	fmt.Printf(">> 读取更新后的消息以确认覆盖\n")
	retrievedUpdated, err := reader.ReadData(username, addr)
	if err != nil {
		fmt.Printf("   读取更新消息失败: %v\n", err)
		log.Printf("   读取更新消息失败: %v\n", err)
		return
	}

	// 验证更新消息
	originalTruncated := ensureMessageLength(originalMessage)
	updatedTruncated := ensureMessageLength(updatedMessage)

	// 检查原始消息和更新消息是否不同
	originalDifferent := !compareByteArrays(originalTruncated, updatedTruncated)
	// 检查更新后读取的消息是否与更新消息匹配
	updatedMatched := compareByteArrays(updatedTruncated, retrievedUpdated)

	fmt.Printf(">> 覆盖测试结果:\n")
	fmt.Printf("   原始消息和更新消息不同: %s\n", boolToCheckmark(originalDifferent))
	fmt.Printf("   读取的消息与更新消息匹配: %s\n", boolToCheckmark(updatedMatched))
	log.Printf(">> 覆盖测试结果:\n")
	log.Printf("   原始消息(Hex): %X\n", originalTruncated)
	log.Printf("   更新消息(Hex): %X\n", updatedTruncated)
	log.Printf("   读取消息(Hex): %X\n", retrievedUpdated)
	log.Printf("   原始消息和更新消息不同: %s\n", boolToVerifiedStr(originalDifferent))
	log.Printf("   读取的消息与更新消息匹配: %s\n", boolToVerifiedStr(updatedMatched))

	if updatedMatched {
		fmt.Printf("   覆盖测试成功: 旧记录已被覆盖\n")
	} else {
		fmt.Printf("   覆盖测试失败: 消息未正确更新\n")
	}
}

func runDeleteTest(reader *CardReader) {
	fmt.Println("\n=== 删除数据测试 ===")
	log.Printf("\n\n=== 删除数据测试 ===\n")

	// 1. 存储数据 -> 2. 读取数据 -> 3. 删除数据 -> 4. 尝试再次读取（应该失败）-> 5. 重新存储

	username := "delete_test_user"
	addr := []byte("Delete Test Address - This address should be exactly 64 bytes for testing!")
	message := []byte("Test message for delete operation")

	// 1. 存储数据
	fmt.Printf(">> 步骤1: 存储测试数据\n")
	log.Printf(">> 步骤1: 存储测试数据\n")
	log.Printf("   用户名: %s\n", username)
	log.Printf("   地址: %s\n", string(addr))
	log.Printf("   消息: %s\n", string(message))

	err := reader.StoreData(username, addr, message)
	if err != nil {
		fmt.Printf("   存储失败: %v\n", err)
		log.Printf("   存储失败: %v\n", err)
		return
	}
	fmt.Printf("   存储成功\n")

	// 2. 读取数据
	fmt.Printf(">> 步骤2: 读取数据验证\n")
	log.Printf(">> 步骤2: 读取数据验证\n")

	retrievedMessage, err := reader.ReadData(username, addr)
	if err != nil {
		fmt.Printf("   读取失败: %v\n", err)
		log.Printf("   读取失败: %v\n", err)
		return
	}

	// 验证读取的数据
	originalTruncated := ensureMessageLength(message)
	matched := compareByteArrays(originalTruncated, retrievedMessage)
	fmt.Printf("   读取成功，数据验证: %s\n", boolToCheckmark(matched))
	log.Printf("   读取成功，数据验证: %s\n", boolToVerifiedStr(matched))

	// 3. 删除数据
	fmt.Printf(">> 步骤3: 删除数据\n")
	log.Printf(">> 步骤3: 删除数据\n")

	err = reader.DeleteData(username, addr)
	if err != nil {
		fmt.Printf("   删除失败: %v\n", err)
		log.Printf("   删除失败: %v\n", err)
		return
	}
	fmt.Printf("   删除成功\n")

	// 4. 尝试再次读取（应该失败）
	fmt.Printf(">> 步骤4: 尝试读取已删除数据（应该失败）\n")
	log.Printf(">> 步骤4: 尝试读取已删除数据（应该失败）\n")

	_, err = reader.ReadData(username, addr)
	if err == nil {
		fmt.Printf("   错误: 成功读取了已删除的数据!\n")
		log.Printf("   错误: 成功读取了已删除的数据!\n")
		return
	}
	fmt.Printf("   预期的读取失败: %v\n", err)
	log.Printf("   预期的读取失败: %v\n", err)

	// 5. 重新存储数据
	fmt.Printf(">> 步骤5: 重新存储数据到已删除的位置\n")
	log.Printf(">> 步骤5: 重新存储数据到已删除的位置\n")

	newMessage := []byte("New message after deletion")
	err = reader.StoreData(username, addr, newMessage)
	if err != nil {
		fmt.Printf("   重新存储失败: %v\n", err)
		log.Printf("   重新存储失败: %v\n", err)
		return
	}
	fmt.Printf("   重新存储成功\n")

	// 6. 再次读取以验证
	fmt.Printf(">> 步骤6: 读取重新存储的数据\n")
	log.Printf(">> 步骤6: 读取重新存储的数据\n")

	retrievedNewMessage, err := reader.ReadData(username, addr)
	if err != nil {
		fmt.Printf("   读取失败: %v\n", err)
		log.Printf("   读取失败: %v\n", err)
		return
	}

	// 验证读取的新数据
	newMessageTruncated := ensureMessageLength(newMessage)
	newMatched := compareByteArrays(newMessageTruncated, retrievedNewMessage)
	fmt.Printf("   读取成功，新数据验证: %s\n", boolToCheckmark(newMatched))
	log.Printf("   读取成功，新数据验证: %s\n", boolToVerifiedStr(newMatched))

	fmt.Printf("\n>> 删除测试完成: %s\n", boolToCheckmark(matched && newMatched && err == nil))
}

func runCapacityTest(reader *CardReader, count int) {
	fmt.Printf("\n=== 容量测试 (尝试存储 %d 条记录) ===\n", count)
	log.Printf("\n\n=== 容量测试 (尝试存储 %d 条记录) ===\n", count)

	// 为了保证唯一性，创建随机用户名和地址
	successCount := 0
	failCount := 0
	startTime := time.Now()

	for i := 0; i < count; i++ {
		// 创建唯一用户名和随机地址
		username := fmt.Sprintf("capacity_test_user_%d_%s", i, hex.EncodeToString(generateRandomMessage(4)))
		addr := generateRandomAddress(32)    // 生成32字节随机地址，会自动填充到64字节
		message := generateRandomMessage(16) // 生成16字节随机消息，会自动填充到32字节

		progress := float64(i+1) / float64(count) * 100
		fmt.Printf("\r进度: %.1f%% (%d/%d)", progress, i+1, count)

		err := reader.StoreData(username, addr, message)
		if err != nil {
			failCount++
			// 只打印第一个错误和每10个错误中的1个
			if failCount == 1 || failCount%10 == 0 {
				fmt.Printf("\n存储失败: %v\n", err)
				log.Printf("记录 #%d 存储失败: %v\n", i, err)
			}
			if strings.Contains(err.Error(), "存储空间已满") {
				fmt.Printf("\n达到芯片存储限制: %d 条记录\n", successCount)
				log.Printf("达到芯片存储限制: %d 条记录\n", successCount)
				break
			}
		} else {
			successCount++
			// 每存储10条记录，随机检查一条
			if successCount%10 == 0 {
				retrievedMessage, err := reader.ReadData(username, addr)
				if err != nil {
					log.Printf("读取验证失败 (记录 #%d): %v\n", i, err)
				} else {
					originalTruncated := ensureMessageLength(message)
					matched := compareByteArrays(originalTruncated, retrievedMessage)
					log.Printf("记录 #%d 验证: %s\n", i, boolToVerifiedStr(matched))
				}
			}
		}
	}

	duration := time.Since(startTime)
	fmt.Printf("\n\n容量测试结果:\n")
	fmt.Printf("  成功存储: %d 条记录\n", successCount)
	fmt.Printf("  失败: %d 尝试\n", failCount)
	fmt.Printf("  总耗时: %v\n", duration)
	fmt.Printf("  平均每条记录: %v\n", duration/time.Duration(successCount+failCount))

	log.Printf("\n容量测试结果:\n")
	log.Printf("  成功存储: %d 条记录\n", successCount)
	log.Printf("  失败: %d 尝试\n", failCount)
	log.Printf("  总耗时: %v\n", duration)
	log.Printf("  平均每条记录: %v\n", duration/time.Duration(successCount+failCount))
}

func runRandomTest(reader *CardReader, count int) {
	fmt.Printf("\n=== 随机数据测试 (测试 %d 条随机记录) ===\n", count)
	log.Printf("\n\n=== 随机数据测试 (测试 %d 条随机记录) ===\n", count)

	rand.Seed(time.Now().UnixNano())
	successCount := 0
	failCount := 0
	readSuccessCount := 0
	readFailCount := 0

	// 保存测试记录以便后续读取验证
	type testRecord struct {
		username string
		addr     []byte
		message  []byte
	}
	records := make([]testRecord, 0, count)

	// 存储随机数据
	fmt.Println(">> 存储随机数据...")
	for i := 0; i < count; i++ {
		username := fmt.Sprintf("random_test_user_%d_%x", i, rand.Uint32())
		addrLen := rand.Intn(64) + 1 // 1-64字节
		addr := generateRandomAddress(addrLen)
		messageLen := rand.Intn(32) + 1 // 1-32字节
		message := generateRandomMessage(messageLen)

		err := reader.StoreData(username, addr, message)
		if err != nil {
			failCount++
			log.Printf("随机记录 #%d 存储失败: %v\n", i, err)
		} else {
			successCount++
			records = append(records, testRecord{username, addr, message})
		}

		progress := float64(i+1) / float64(count) * 100
		fmt.Printf("\r存储进度: %.1f%% (%d/%d)", progress, i+1, count)
	}
	fmt.Println() // 换行

	// 读取并验证
	fmt.Println(">> 读取并验证随机数据...")
	for i, record := range records {
		retrievedMessage, err := reader.ReadData(record.username, record.addr)
		if err != nil {
			readFailCount++
			log.Printf("随机记录 #%d 读取失败: %v\n", i, err)
		} else {
			originalTruncated := ensureMessageLength(record.message)
			matched := compareByteArrays(originalTruncated, retrievedMessage)
			if matched {
				readSuccessCount++
			} else {
				readFailCount++
				log.Printf("随机记录 #%d 验证失败！原始: %X, 读取: %X\n",
					i, originalTruncated, retrievedMessage)
			}
		}

		progress := float64(i+1) / float64(len(records)) * 100
		fmt.Printf("\r读取验证进度: %.1f%% (%d/%d)", progress, i+1, len(records))
	}
	fmt.Println() // 换行

	fmt.Printf("\n随机测试结果:\n")
	fmt.Printf("  存储成功: %d/%d (%.1f%%)\n",
		successCount, count, float64(successCount)/float64(count)*100)
	fmt.Printf("  读取验证成功: %d/%d (%.1f%%)\n",
		readSuccessCount, len(records), float64(readSuccessCount)/float64(len(records))*100)

	log.Printf("\n随机测试结果:\n")
	log.Printf("  存储成功: %d/%d (%.1f%%)\n",
		successCount, count, float64(successCount)/float64(count)*100)
	log.Printf("  读取验证成功: %d/%d (%.1f%%)\n",
		readSuccessCount, len(records), float64(readSuccessCount)/float64(len(records))*100)
}

// 比较两个字节数组是否相等
func compareByteArrays(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// 辅助函数: bool转对勾/叉符号
func boolToCheckmark(b bool) string {
	if b {
		return "✓ 匹配"
	}
	return "✗ 不匹配"
}

// 辅助函数: bool转验证状态字符串
func boolToVerifiedStr(b bool) string {
	if b {
		return "验证通过 - 内容完全匹配"
	}
	return "验证失败 - 内容不匹配"
}

func runTests() error {
	// 初始化日志
	now := time.Now().Format("2006-01-02_150405")
	logFile, err := os.Create(fmt.Sprintf("applet_test_%s.log", now))
	if err != nil {
		return fmt.Errorf("无法创建日志文件: %v", err)
	}
	defer logFile.Close()

	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	log.Printf("=== 安全芯片Applet测试开始 - %s ===\n", time.Now().Format("2006-01-02 15:04:05"))

	// 连接到智能卡
	reader, err := NewCardReader()
	if err != nil {
		return fmt.Errorf("初始化读卡器失败: %v", err)
	}
	defer reader.Close()

	// 连接到读卡器
	if err := reader.Connect(""); err != nil {
		return fmt.Errorf("连接读卡器失败: %v", err)
	}

	// 选择Applet
	if err := reader.SelectApplet(); err != nil {
		return fmt.Errorf("选择Applet失败: %v", err)
	}

	// 运行基本测试
	runBasicTests(reader)

	// 运行覆盖测试
	runOverwriteTest(reader)

	// 运行删除测试
	runDeleteTest(reader)

	// 运行随机数据测试
	runRandomTest(reader, 20)

	// 运行容量测试
	runCapacityTest(reader, 110) // 尝试存储超过限制的记录

	log.Printf("=== 安全芯片Applet测试结束 - %s ===\n", time.Now().Format("2006-01-02 15:04:05"))
	return nil
}

func main() {
	fmt.Println("=== 安全芯片Applet测试工具 ===")
	fmt.Println("本工具将测试JavaCard安全芯片的数据存储和读取功能")
	fmt.Println("详细执行记录将保存在日志文件中")

	rand.Seed(time.Now().UnixNano())

	err := runTests()
	if err != nil {
		fmt.Printf("测试出错: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n所有测试已完成!")
}
