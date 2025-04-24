package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"regexp"
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
	return ciphertext, nil
}

// DecryptAES 使用AES-GCM解密数据
func DecryptAES(ciphertext []byte, key []byte) ([]byte, error) {
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
func ExtractEthAddress(publicKey string) (string, error) {
	// 检查公钥格式是否正确
	match, err := regexp.MatchString(`^0[234][0-9A-Fa-f]{64}$`, publicKey)
	if err != nil || !match {
		return "", errors.New("公钥格式不正确")
	}

	// 去掉前缀（0x或04）
	pubBytes, err := hex.DecodeString(publicKey[2:])
	if err != nil {
		return "", err
	}

	// Keccak-256哈希
	hasher := sha256.New()
	hasher.Write(pubBytes)
	hash := hasher.Sum(nil)

	// 取最后20字节作为以太坊地址
	address := "0x" + hex.EncodeToString(hash[12:])

	return address, nil
}
