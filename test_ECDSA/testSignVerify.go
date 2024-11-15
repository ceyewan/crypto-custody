package main

// TestS ECDSA-secp256k1 Sign and Verify

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
	r, _ := hex.DecodeString("af6a40b60e64a71d4729f97392e02f35f704aadd1e7401988f7ce79890ab0495")
	s, _ := hex.DecodeString("3bfd7272033dfd9f2822172cb97ec8382c393599ceb9ea282a6c4779ce76408b")
	signature := append(r, s...)
	fmt.Println("签名:", signature)

	// 构造公钥
	x, _ := new(big.Int).SetString("202a222563e6cf5000389edf277a8a81c50c1025e9f9ac98d97689356f5e99e3", 16)
	y, _ := new(big.Int).SetString("d4341455d382d9467fa49c7bd9e1c9f311d3eadf8d17b8af5bb4969f6b164d06", 16)
	PublicKey := &ecdsa.PublicKey{Curve: crypto.S256(), X: x, Y: y}
	fmt.Println("公钥:", PublicKey)

	// 验证签名
	publicKeyBytes := crypto.FromECDSAPub(PublicKey)
	valid := crypto.VerifySignature(publicKeyBytes, digest, signature)
	fmt.Printf("签名验证结果: %v\n", valid)
}
