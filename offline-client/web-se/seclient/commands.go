package seclient

import (
	"encoding/hex"
	"fmt"
)

// StoreData 存储数据 - 简化接口，直接接收数据
func (r *CardReader) StoreData(username []byte, addr []byte, message []byte) (byte, byte, error) {
	// 验证输入数据长度
	if len(username) != USERNAME_LENGTH {
		return 0, 0, fmt.Errorf("用户名长度错误: 应为 %d 字节", USERNAME_LENGTH)
	}
	if len(addr) != ADDR_LENGTH {
		return 0, 0, fmt.Errorf("地址长度错误: 应为 %d 字节", ADDR_LENGTH)
	}
	if len(message) != MESSAGE_LENGTH {
		return 0, 0, fmt.Errorf("消息长度错误: 应为 %d 字节", MESSAGE_LENGTH)
	}

	// 构造完整数据
	fullData := make([]byte, 0, USERNAME_LENGTH+ADDR_LENGTH+MESSAGE_LENGTH)
	fullData = append(fullData, username...)
	fullData = append(fullData, addr...)
	fullData = append(fullData, message...)

	fmt.Println("❕存储数据到安全芯片❕")
	fmt.Println("username:", hex.EncodeToString(username))
	fmt.Println("addr:", hex.EncodeToString(addr))
	fmt.Println("message:", hex.EncodeToString(message))

	// 构建APDU命令
	command := []byte{CLA, INS_STORE_DATA, 0x00, 0x00, byte(len(fullData))}
	command = append(command, fullData...)

	// 发送命令
	data, sw, err := r.TransmitAPDU(command)
	if err != nil {
		return 0, 0, err
	}

	// 处理响应状态
	if sw != SW_SUCCESS {
		if sw == SW_FILE_FULL {
			return 0, 0, fmt.Errorf("存储失败: 存储空间已满 (状态码: 0x%04X)", sw)
		} else if sw == SW_WRONG_LENGTH {
			return 0, 0, fmt.Errorf("存储失败: 数据长度错误 (状态码: 0x%04X)", sw)
		}
		return 0, 0, fmt.Errorf("存储失败: 未知错误 (状态码: 0x%04X)", sw)
	}

	// 解析响应数据
	if len(data) < 2 {
		return 0, 0, fmt.Errorf("响应数据不完整")
	}

	recordIndex := data[0]
	recordCount := data[1]

	return recordIndex, recordCount, nil
}

// ReadData 读取数据 - 简化接口，接收外部生成的签名
func (r *CardReader) ReadData(username []byte, addr []byte, signature []byte) ([]byte, error) {
	// 验证输入数据长度
	if len(username) != USERNAME_LENGTH {
		return nil, fmt.Errorf("用户名长度错误: 应为 %d 字节", USERNAME_LENGTH)
	}
	if len(addr) != ADDR_LENGTH {
		return nil, fmt.Errorf("地址长度错误: 应为 %d 字节", ADDR_LENGTH)
	}
	if len(signature) < 8 || len(signature) > MAX_SIGNATURE_LENGTH {
		return nil, fmt.Errorf("签名长度错误: 应为 8-%d 字节", MAX_SIGNATURE_LENGTH)
	}

	// 构造完整数据 - 包含签名
	fullData := make([]byte, 0, USERNAME_LENGTH+ADDR_LENGTH+len(signature))
	fullData = append(fullData, username...)
	fullData = append(fullData, addr...)
	fullData = append(fullData, signature...)

	fmt.Println("❕读取数据从安全芯片❕")
	fmt.Println("username:", hex.EncodeToString(username))
	fmt.Println("addr:", hex.EncodeToString(addr))
	fmt.Println("signature:", hex.EncodeToString(signature))

	// 构建APDU命令
	command := []byte{CLA, INS_READ_DATA, 0x00, 0x00, byte(len(fullData))}
	command = append(command, fullData...)

	// 发送命令
	data, sw, err := r.TransmitAPDU(command)
	if err != nil {
		return nil, err
	}

	// 处理响应状态
	if sw == SW_RECORD_NOT_FOUND {
		return nil, fmt.Errorf("记录未找到 (状态码: 0x%04X)", sw)
	} else if sw == SW_SIGNATURE_INVALID {
		return nil, fmt.Errorf("签名无效 (状态码: 0x%04X)", sw)
	} else if sw != SW_SUCCESS {
		return nil, fmt.Errorf("读取数据返回错误状态码: 0x%04X", sw)
	}

	fmt.Println("❕读取数据成功❕")
	fmt.Println("数据:", hex.EncodeToString(data))

	return data, nil
}

// DeleteData 删除数据 - 简化接口，接收外部生成的签名
func (r *CardReader) DeleteData(username []byte, addr []byte, signature []byte) (byte, byte, error) {
	// 验证输入数据长度
	if len(username) != USERNAME_LENGTH {
		return 0, 0, fmt.Errorf("用户名长度错误: 应为 %d 字节", USERNAME_LENGTH)
	}
	if len(addr) != ADDR_LENGTH {
		return 0, 0, fmt.Errorf("地址长度错误: 应为 %d 字节", ADDR_LENGTH)
	}
	if len(signature) < 8 || len(signature) > MAX_SIGNATURE_LENGTH {
		return 0, 0, fmt.Errorf("签名长度错误: 应为 8-%d 字节", MAX_SIGNATURE_LENGTH)
	}

	// 构造完整数据 - 包含签名
	fullData := make([]byte, 0, USERNAME_LENGTH+ADDR_LENGTH+len(signature))
	fullData = append(fullData, username...)
	fullData = append(fullData, addr...)
	fullData = append(fullData, signature...)

	// 构建APDU命令
	command := []byte{CLA, INS_DELETE_DATA, 0x00, 0x00, byte(len(fullData))}
	command = append(command, fullData...)

	// 发送命令
	data, sw, err := r.TransmitAPDU(command)
	if err != nil {
		return 0, 0, err
	}

	// 处理响应状态
	if sw == SW_RECORD_NOT_FOUND {
		return 0, 0, fmt.Errorf("记录未找到 (状态码: 0x%04X)", sw)
	} else if sw == SW_SIGNATURE_INVALID {
		return 0, 0, fmt.Errorf("签名无效 (状态码: 0x%04X)", sw)
	} else if sw != SW_SUCCESS {
		return 0, 0, fmt.Errorf("删除数据返回错误状态码: 0x%04X", sw)
	}

	// 解析响应数据
	if len(data) < 2 {
		return 0, 0, fmt.Errorf("响应数据不完整")
	}

	recordIndex := data[0]
	remainingCount := data[1]

	return recordIndex, remainingCount, nil
}
