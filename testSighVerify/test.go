package main

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
)

func main() {
	// 消息哈希
	digest, _ := hex.DecodeString("4cca684310ba248350490ab3e82363f147e0ab7c3ff78454c1b53d04b3dab60a")
	fmt.Println("消息哈希:", digest)

	// 构造签名
	r, _ := hex.DecodeString("13458f243cb14e37fbda8e382fb1161bee00d75f05dc1eac345a1c7b6940f34c")
	s, _ := hex.DecodeString("301794f78e33a2d5d315ddefc5c8c99c017fb7148826e68d8e1eb410a1acafb1")
	signature := append(r, s...)
	fmt.Println("签名:", signature)

	// 构造公钥
	x, _ := new(big.Int).SetString("fa99b551434767153242641b43737a17ac1725ef19470d3984635914e64ba302", 16)
	y, _ := new(big.Int).SetString("1e49c568ce3a0dab1e3a0430769376840bc6937b3055f6ddd2ed1d97105fbf6c", 16)
	PublicKey := &ecdsa.PublicKey{Curve: crypto.S256(), X: x, Y: y}
	fmt.Println("公钥:", PublicKey)

	// 验证签名
	publicKeyBytes := crypto.FromECDSAPub(PublicKey)
	valid := crypto.VerifySignature(publicKeyBytes, digest, signature)
	fmt.Printf("签名验证结果: %v\n", valid)
}
