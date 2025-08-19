package utils

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
)

// EcdsaSignature 表示提供的签名格式
type EcdsaSignature struct {
	R struct {
		Curve  string  `json:"curve"`
		Scalar []int64 `json:"scalar"`
	} `json:"r"`
	S struct {
		Curve  string  `json:"curve"`
		Scalar []int64 `json:"scalar"`
	} `json:"s"`
	Recid int `json:"recid"`
}

// ConvertToEthSignature 将提供的签名格式转换为以太坊交易签名格式
// 返回65字节的签名数据，格式为: r (32字节) + s (32字节) + v (1字节)
func ConvertToEthSignature(signatureJSON string) (string, error) {
	var signature EcdsaSignature
	err := json.Unmarshal([]byte(signatureJSON), &signature)
	if err != nil {
		return "", fmt.Errorf("解析签名JSON失败: %v", err)
	}

	// 确保使用的是secp256k1曲线
	if signature.R.Curve != "secp256k1" {
		return "", fmt.Errorf("签名必须使用secp256k1曲线")
	}

	// 将r和s值转换为大整数
	rInt := scalarToBigInt(signature.R.Scalar)
	sInt := scalarToBigInt(signature.S.Scalar)

	// 生成以太坊格式的签名
	// 以太坊签名是65字节: r (32字节) + s (32字节) + v (1字节)
	rBytes := padTo32Bytes(rInt.Bytes())
	sBytes := padTo32Bytes(sInt.Bytes())

	// 以太坊的v值通常是 recid + 27
	v := byte(signature.Recid + 27)

	// 组合成完整的签名
	ethSignature := append(append(rBytes, sBytes...), v)

	// 返回十六进制格式的签名
	return "0x" + hex.EncodeToString(ethSignature), nil
}

// 将标量数组转换为大整数
func scalarToBigInt(scalar []int64) *big.Int {
	result := new(big.Int)

	// 创建一个字节数组
	bytes := make([]byte, len(scalar))
	for i, val := range scalar {
		bytes[i] = byte(val)
	}

	// 将字节数组转换为大整数
	return result.SetBytes(bytes)
}

// 将字节数组填充到32字节
func padTo32Bytes(input []byte) []byte {
	if len(input) >= 32 {
		return input
	}

	result := make([]byte, 32)
	// 从右向左填充，保留前导零
	copy(result[32-len(input):], input)
	return result
}
