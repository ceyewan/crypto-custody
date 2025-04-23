package seclient

import (
	"crypto/sha256"
)

// 将用户名转换为固定长度的字节数组
func usernameToBytes(username string) []byte {
	// 使用SHA256哈希算法将任意长度的用户名映射为固定32字节
	hash := sha256.Sum256([]byte(username))
	return hash[:]
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
