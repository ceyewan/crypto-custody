package seclient

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// ECDSA签名函数 - 生成DER格式的签名
func signData(privateKey *ecdsa.PrivateKey, data []byte) ([]byte, error) {
	// 计算数据哈希
	h := sha256.Sum256(data)

	// 使用ECDSA对哈希签名，直接获取DER格式签名
	signature, err := privateKey.Sign(nil, h[:], nil)
	if err != nil {
		return nil, fmt.Errorf("签名失败: %v", err)
	}

	return signature, nil
}

// 从PEM文件中加载ECDSA私钥
func loadPrivateKeyFromPEM(filePath string) (*ecdsa.PrivateKey, error) {
	// 读取私钥文件
	pemData, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取私钥文件失败: %v", err)
	}

	// 解码PEM数据
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("解码PEM数据失败")
	}

	// 解析私钥
	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("解析ECDSA私钥失败: %v", err)
	}

	return key, nil
}

// 准备签名数据 - 对用户名和地址数据进行签名
func prepareSignature(username string, addr []byte, privateKeyPath string) ([]byte, error) {
	// 加载私钥
	privateKey, err := loadPrivateKeyFromPEM(privateKeyPath)
	if err != nil {
		return nil, err
	}

	// 处理数据
	usernameBytes := usernameToBytes(username)
	addrBytes := ensureAddrLength(addr)

	// 组合数据并签名
	dataToSign := append(usernameBytes, addrBytes...)
	signature, err := signData(privateKey, dataToSign)
	if err != nil {
		return nil, err
	}

	return signature, nil
}
