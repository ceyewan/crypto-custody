package seclient

import (
	"fmt"
	"log"
)

// assertLength 验证输入长度是否符合要求
func assertLength(value []byte, expectedLen int, name string) error {
	if len(value) != expectedLen {
		log.Printf("错误: %s 长度错误: 应为 %d 字节, 实际为 %d 字节", name, expectedLen, len(value))
		return fmt.Errorf("%s 长度错误: 应为 %d 字节, 实际为 %d 字节", name, expectedLen, len(value))
	}
	return nil
}

// assertSignatureLength 验证签名长度是否符合要求（允许范围）
func assertSignatureLength(sign []byte, minLen, maxLen int) error {
	if len(sign) < minLen || len(sign) > maxLen {
		log.Printf("错误: 签名长度错误: 应为 %d-%d 字节, 实际为 %d 字节", minLen, maxLen, len(sign))
		return fmt.Errorf("签名长度错误: 应为 %d-%d 字节, 实际为 %d 字节", minLen, maxLen, len(sign))
	}
	return nil
}

// StoreData 存储数据 - 符合APDU格式
func (r *CardReader) StoreData(username string, addr []byte, message []byte) error {
	// 通过assert函数验证输入
	usernameBytes := usernameToBytes(username)
	if err := assertLength(usernameBytes, USERNAME_LENGTH, "用户名"); err != nil {
		return err
	}
	if err := assertLength(addr, ADDR_LENGTH, "地址"); err != nil {
		return err
	}
	if err := assertLength(message, MESSAGE_LENGTH, "消息"); err != nil {
		return err
	}

	// 构造完整数据
	fullData := make([]byte, 0, USERNAME_LENGTH+ADDR_LENGTH+MESSAGE_LENGTH)
	fullData = append(fullData, usernameBytes...)
	fullData = append(fullData, addr...)
	fullData = append(fullData, message...)

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
	log.Printf("    地址(前16字节): %X...\n", addr[:min(16, len(addr))])
	log.Printf("    消息(前16字节): %X...\n", message[:min(16, len(message))])

	// 发送命令
	resp, err := r.card.Transmit(command)
	if err != nil {
		log.Printf("错误: 发送存储数据命令失败: %v", err)
		return fmt.Errorf("发送存储数据命令失败: %v", err)
	}

	sw, data := extractResponseAndSW(resp)
	if sw != SW_SUCCESS {
		if sw == SW_FILE_FULL {
			log.Printf("错误: 存储失败: 存储空间已满 (状态码: 0x%04X)", sw)
			return fmt.Errorf("存储失败: 存储空间已满 (状态码: 0x%04X)", sw)
		} else if sw == SW_WRONG_LENGTH {
			log.Printf("错误: 存储失败: 数据长度错误 (状态码: 0x%04X)", sw)
			return fmt.Errorf("存储失败: 数据长度错误 (状态码: 0x%04X)", sw)
		}
		log.Printf("错误: 存储失败: 未知错误 (状态码: 0x%04X)", sw)
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

// ReadData 读取数据 - 符合APDU格式，添加签名参数
func (r *CardReader) ReadData(username string, addr []byte, sign []byte) ([]byte, error) {
	// 通过assert函数验证输入
	usernameBytes := usernameToBytes(username)
	if err := assertLength(usernameBytes, USERNAME_LENGTH, "用户名"); err != nil {
		return nil, err
	}
	if err := assertLength(addr, ADDR_LENGTH, "地址"); err != nil {
		return nil, err
	}
	if err := assertSignatureLength(sign, 70, 72); err != nil {
		return nil, err
	}

	// 构造完整数据 - 增加签名部分
	fullData := make([]byte, 0, USERNAME_LENGTH+ADDR_LENGTH+len(sign))
	fullData = append(fullData, usernameBytes...)
	fullData = append(fullData, addr...)
	fullData = append(fullData, sign...)

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
	log.Printf("  Data: [用户名(32字节)][地址(64字节)][签名(%d字节)]\n", len(sign))
	log.Printf("    用户名(哈希值): %X\n", usernameBytes[:min(16, len(usernameBytes))])
	log.Printf("    地址(前16字节): %X...\n", addr[:min(16, len(addr))])
	log.Printf("    签名(前16字节): %X...\n", sign[:min(16, len(sign))])

	// 发送命令
	resp, err := r.card.Transmit(command)
	if err != nil {
		log.Printf("错误: 发送读取数据命令失败: %v", err)
		return nil, fmt.Errorf("发送读取数据命令失败: %v", err)
	}

	sw, data := extractResponseAndSW(resp)
	if sw == SW_RECORD_NOT_FOUND {
		log.Printf("错误: 记录未找到 (状态码: 0x%04X)", sw)
		return nil, fmt.Errorf("记录未找到 (状态码: 0x%04X)", sw)
	} else if sw != SW_SUCCESS {
		log.Printf("错误: 读取数据返回错误状态码: 0x%04X", sw)
		return nil, fmt.Errorf("读取数据返回错误状态码: 0x%04X", sw)
	}

	log.Printf("\n=== 读取数据响应 ===\n")
	log.Printf("响应数据: %X\n", resp)
	log.Printf("状态码: 0x%04X (成功)\n", sw)
	log.Printf("解析响应: 读取到消息数据 %d 字节\n", len(data))
	log.Printf("消息数据: %X\n", data)

	return data, nil
}

// DeleteData 删除数据 - 根据用户名和地址删除记录，包含签名验证
func (r *CardReader) DeleteData(username string, addr []byte, sign []byte) error {
	// 通过assert函数验证输入
	usernameBytes := usernameToBytes(username)
	if err := assertLength(usernameBytes, USERNAME_LENGTH, "用户名"); err != nil {
		return err
	}
	if err := assertLength(addr, ADDR_LENGTH, "地址"); err != nil {
		return err
	}
	if err := assertSignatureLength(sign, 70, 72); err != nil {
		return err
	}

	// 构造完整数据 - 增加签名部分
	fullData := make([]byte, 0, USERNAME_LENGTH+ADDR_LENGTH+len(sign))
	fullData = append(fullData, usernameBytes...)
	fullData = append(fullData, addr...)
	fullData = append(fullData, sign...)

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
	log.Printf("  Data: [用户名(32字节)][地址(64字节)][签名(%d字节)]\n", len(sign))
	log.Printf("    用户名(哈希值): %X\n", usernameBytes[:min(16, len(usernameBytes))])
	log.Printf("    地址(前16字节): %X...\n", addr[:min(16, len(addr))])
	log.Printf("    签名(前16字节): %X...\n", sign[:min(16, len(sign))])

	// 发送命令
	resp, err := r.card.Transmit(command)
	if err != nil {
		log.Printf("错误: 发送删除数据命令失败: %v", err)
		return fmt.Errorf("发送删除数据命令失败: %v", err)
	}

	sw, data := extractResponseAndSW(resp)
	if sw == SW_RECORD_NOT_FOUND {
		log.Printf("错误: 记录未找到 (状态码: 0x%04X)", sw)
		return fmt.Errorf("记录未找到 (状态码: 0x%04X)", sw)
	} else if sw != SW_SUCCESS {
		log.Printf("错误: 删除数据返回错误状态码: 0x%04X", sw)
		return fmt.Errorf("删除数据返回错误状态码: 0x%04X", sw)
	}

	log.Printf("\n=== 删除数据响应 ===\n")
	log.Printf("响应数据: %X\n", resp)
	log.Printf("状态码: 0x%04X (成功)\n", sw)
	log.Printf("解析响应: 删除的记录索引=%d, 剩余记录数=%d\n", data[0], data[1])

	return nil
}
