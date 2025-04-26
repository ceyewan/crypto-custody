package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"regexp"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/crypto"
)

// GenerateRandomBytes 生成指定长度的随机字节序列
func GenerateRandomBytes(size int) ([]byte, error) {
	bytes := make([]byte, size)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return nil, err
	}
	return bytes, nil
}

// EncryptAES 使用AES-GCM加密数据
func EncryptAES(plaintext []byte, key []byte) ([]byte, error) {
	fmt.Println("加密密钥", hex.EncodeToString(key))
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 从随机数生成器创建一个新的随机数
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 使用AES-GCM模式进行加密，nonce会被添加到密文前面
	ciphertext := aesgcm.Seal(nonce, nonce, plaintext, nil)
	fmt.Println("加密结果", hex.EncodeToString(ciphertext)[:20])
	return ciphertext, nil
}

// DecryptAES 使用AES-GCM解密数据
func DecryptAES(ciphertext []byte, key []byte) ([]byte, error) {
	fmt.Println("解密密钥", hex.EncodeToString(key))
	fmt.Println("解密数据", hex.EncodeToString(ciphertext)[:20])
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < 12 {
		return nil, errors.New("密文长度不正确")
	}

	// 从密文中分离出nonce
	nonce := ciphertext[:12]
	// 实际的密文
	ciphertext = ciphertext[12:]

	// 解密
	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// ExtractEthAddress 从公钥提取以太坊地址
func ExtractEthAddress(publicKeyHex string) (string, error) {
	// 检查公钥格式是否正确（压缩格式：02或03开头，后跟64个十六进制字符）
	match, err := regexp.MatchString(`^0[23][0-9A-Fa-f]{64}$`, publicKeyHex)
	if err != nil || !match {
		return "", errors.New("公钥格式不正确")
	}

	// 解码公钥hex字符串
	pubBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return "", err
	}

	// 使用btcec库解析压缩公钥
	pubKey, err := btcec.ParsePubKey(pubBytes)
	if err != nil {
		return "", errors.New("无效的压缩公钥格式")
	}

	// 转换为以太坊ECDSA公钥
	ecdsaPubKey := &ecdsa.PublicKey{
		Curve: crypto.S256(), // 以太坊使用的secp256k1曲线
		X:     pubKey.X(),
		Y:     pubKey.Y(),
	}

	// 使用以太坊库计算地址（内部会正确使用Keccak-256哈希）
	address := crypto.PubkeyToAddress(*ecdsaPubKey).Hex()

	return address, nil
}
