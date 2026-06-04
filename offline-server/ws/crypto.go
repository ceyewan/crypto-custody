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
	"strings"
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

// SignData 对 SE 授权数据签名。
// recordID 是 32 字节记录编号的 hex 表示；Applet 看到的第一段字段仍为 32 字节。
func SignData(recordID, address string) (string, error) {
	recordBytes, err := hex.DecodeString(strings.TrimPrefix(recordID, "0x"))
	if err != nil || len(recordBytes) != 32 {
		return "", fmt.Errorf("record_id必须是32字节hex")
	}
	addrBytes, err := hex.DecodeString(strings.TrimPrefix(address, "0x"))
	if err != nil || len(addrBytes) != 20 {
		return "", fmt.Errorf("地址格式错误")
	}
	data := append(recordBytes, addrBytes...)
	// 计算消息哈希
	hash := sha256.Sum256(data)

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
	s = normalizeECDSALowS(privateKey, s)

	// 将r和s转换为DER格式
	signature, err := marshalECDSASignature(r, s)
	if err != nil {
		return "", err
	}

	// 转为base64
	return base64.StdEncoding.EncodeToString(signature), nil
}

func normalizeECDSALowS(privateKey *ecdsa.PrivateKey, s *big.Int) *big.Int {
	if privateKey == nil || privateKey.Curve == nil || privateKey.Curve.Params() == nil || s == nil {
		return s
	}
	order := privateKey.Curve.Params().N
	if order == nil {
		return s
	}
	halfOrder := new(big.Int).Rsh(new(big.Int).Set(order), 1)
	if s.Cmp(halfOrder) <= 0 {
		return s
	}
	return new(big.Int).Sub(order, s)
}

// 将ECDSA签名转换为DER格式
func marshalECDSASignature(r, s *big.Int) ([]byte, error) {
	// ASN.1格式的签名结构
	type ecdsaSignature struct {
		R, S *big.Int
	}
	return asn1.Marshal(ecdsaSignature{R: r, S: s})
}
