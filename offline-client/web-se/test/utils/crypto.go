package utils

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
)

// 从PEM文件加载私钥
func LoadPrivateKey(filename string) (*ecdsa.PrivateKey, error) {
	pemData, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取私钥文件失败: %v", err)
	}

	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("无法解析PEM数据")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("解析私钥失败: %v", err)
	}

	ecdsaKey, ok := privateKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("私钥不是ECDSA类型")
	}

	return ecdsaKey, nil
}

// 从JSON文件加载密钥生成结果
func LoadKeyGenResult(filename string) (*KeygenResponse, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取密钥生成结果文件失败: %v", err)
	}

	var result KeygenResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("解析JSON数据失败: %v", err)
	}

	// 如果文件中没有用户名，则从文件名生成一个
	if result.UserName == "" {
		baseName := filepath.Base(filename)
		result.UserName = fmt.Sprintf("user_%s", baseName)
	}

	return &result, nil
}

// 对数据进行签名
func SignData(privateKey *ecdsa.PrivateKey, data []byte) (string, error) {
	// 计算消息哈希
	hash := sha256.Sum256(data)

	// 签名
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		return "", fmt.Errorf("签名失败: %v", err)
	}

	// 将r和s转换为DER格式
	signature, err := marshalECDSASignature(r, s)
	if err != nil {
		return "", err
	}

	// 转为base64
	return base64.StdEncoding.EncodeToString(signature), nil
}

// 将ECDSA签名转换为DER格式
func marshalECDSASignature(r, s *big.Int) ([]byte, error) {
	// ASN.1格式的签名结构
	type ecdsaSignature struct {
		R, S *big.Int
	}
	return asn1.Marshal(ecdsaSignature{R: r, S: s})
}
