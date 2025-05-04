package ws

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
)

// 从PEM文件加载私钥
func loadPrivateKey(filename string) (*ecdsa.PrivateKey, error) {
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

// 对数据进行签名
func SignData(username, address string) (string, error) {
	hash := sha256.Sum256([]byte(username))
	userBytes := hash[:]
	addrBytes, _ := hex.DecodeString(address[2:])
	data := append(userBytes, addrBytes...)
	// 计算消息哈希
	hash = sha256.Sum256(data)

	// 加载私钥
	privateKey, err := loadPrivateKey("./private_keys/ec_private_key.pem")
	if err != nil {
		return "", fmt.Errorf("加载私钥失败: %v", err)
	}

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
