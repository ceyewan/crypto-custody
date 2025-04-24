package seclient

// 辅助函数: 从APDU响应中提取状态码和数据
func extractResponseAndSW(resp []byte) (uint16, []byte) {
	if len(resp) < 2 {
		return 0, nil
	}
	sw := (uint16(resp[len(resp)-2]) << 8) | uint16(resp[len(resp)-1])
	data := resp[:len(resp)-2]
	return sw, data
}
