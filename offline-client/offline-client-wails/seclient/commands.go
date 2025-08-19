package seclient

import (
	"encoding/hex"
	"fmt"
	"offline-client-wails/clog"
)

// StoreData 存储数据 - 简化接口，直接接收数据
func (r *CardReader) StoreData(username []byte, addr []byte, message []byte) (int, int, error) {
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

	if r.debug {
		clog.Info("❕存储数据到安全芯片❕",
			clog.String("username", hex.EncodeToString(username)),
			clog.String("addr", hex.EncodeToString(addr)),
			clog.String("message", hex.EncodeToString(message)),
		)
	}

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

	recordIndex := int(data[0])
	recordCount := int(data[1])

	if r.debug {
		clog.Info("❕存储数据成功❕",
			clog.Int("记录索引", recordIndex),
			clog.Int("记录总数", recordCount),
		)
	}

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

	if r.debug {
		clog.Info("❕读取数据从安全芯片❕",
			clog.String("username", hex.EncodeToString(username)),
			clog.String("addr", hex.EncodeToString(addr)),
			clog.String("signature", hex.EncodeToString(signature)),
		)
	}

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

	if r.debug {
		clog.Info("❕读取数据成功❕")
		clog.Infof("数据: %s", hex.EncodeToString(data))
	}

	return data, nil
}

// DeleteData 删除数据 - 简化接口，接收外部生成的签名
func (r *CardReader) DeleteData(username []byte, addr []byte, signature []byte) (int, int, error) {
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

	if r.debug {
		clog.Info("❕删除数据从安全芯片❕",
			clog.String("username", hex.EncodeToString(username)),
			clog.String("addr", hex.EncodeToString(addr)),
			clog.String("signature", hex.EncodeToString(signature)),
		)
	}

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

	recordIndex := int(data[0])
	remainingCount := int(data[1])

	if r.debug {
		clog.Info("❕删除数据成功❕",
			clog.Int("记录索引", recordIndex),
			clog.Int("剩余记录数", remainingCount))
	}

	return recordIndex, remainingCount, nil
}

// GetCPLC 获取CPLC数据
func (r *CardReader) GetCPLC() ([]byte, error) {
	if r.cplcData != nil {
		return r.cplcData, nil
	}

	// 如果缓存中没有，尝试重新获取
	if r.debug {
		clog.Debug("CPLC数据未缓存，尝试获取")
	}

	data, sw, err := r.TransmitAPDU(GET_CPLC_APDU)
	if err != nil {
		return nil, fmt.Errorf("获取CPLC数据失败: %v", err)
	}

	if sw != SW_SUCCESS {
		return nil, fmt.Errorf("获取CPLC数据返回错误状态码: 0x%04X", sw)
	}

	// 验证CPLC数据格式: 期望格式为 [9F7F + 长度字节 + CPLC数据]
	if len(data) < 3 {
		return nil, fmt.Errorf("CPLC数据长度不足")
	}

	// 验证标签是否为 9F7F (两字节)
	if data[0] != 0x9F || data[1] != 0x7F {
		return nil, fmt.Errorf("CPLC数据标签错误: 期望 9F7F, 实际 %02X%02X", data[0], data[1])
	}

	// 读取长度字节
	length := int(data[2])

	// 验证数据长度是否一致
	if len(data) != 3+length {
		return nil, fmt.Errorf("CPLC数据长度不匹配: 标记长度 %d, 实际数据长度 %d", length, len(data)-3)
	}

	// 提取CPLC数据
	cplcData := data[3 : 3+length]

	// 更新缓存
	r.cplcData = cplcData
	clog.Debug("获取CPLC数据成功",
		clog.String("CPLC标签", "9F7F"),
		clog.Int("CPLC长度", length),
		clog.String("CPLC数据", hex.EncodeToString(cplcData)))
	return cplcData, nil
}
