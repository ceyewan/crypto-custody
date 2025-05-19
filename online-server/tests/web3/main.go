package main

import (
	"bufio"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// 全局变量，用于存储账户地址到私钥的映射
var addressToPrivateKey = make(map[string]string)

// 创建钱包
func createWallet() string {
	// 生成一个新的私钥
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}
	// 从私钥中获取公钥
	publicKey := privateKey.Public()

	// 从公钥中获取地址
	address := crypto.PubkeyToAddress(*publicKey.(*ecdsa.PublicKey)).Hex()
	// 从私钥中获取私钥字符串
	privateKeyStr := hex.EncodeToString(privateKey.D.Bytes())

	// 分 X 和 Y 两个部分打印公钥
	// fmt.Printf("X: %x\n", publicKey.(*ecdsa.PublicKey).X)
	// fmt.Printf("Y: %x\n", publicKey.(*ecdsa.PublicKey).Y)
	// fmt.Printf("地址: %s\n", address)
	// fmt.Printf("私钥: %s\n", privateKeyStr)

	// 将地址和私钥保存到映射中
	addressToPrivateKey[address] = privateKeyStr

	return address
}

func signData(data string, address string) string {
	// 查找对应的私钥
	privateKeyHex, exists := addressToPrivateKey[address]
	if !exists {
		log.Fatalf("找不到地址 %s 对应的私钥", address)
	}

	// 生成私钥
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatal(err)
	}

	// data 转 common.Hash
	var hash common.Hash

	// 移除可能的"0x"前缀
	data = strings.TrimPrefix(data, "0x")

	hashBytes, err := hex.DecodeString(data)
	if err != nil {
		log.Fatalf("无效的十六进制数据: %v", err)
	}

	copy(hash[:], hashBytes)

	// 签名数据
	signature, err := crypto.Sign(hash[:], privateKey)
	if err != nil {
		log.Fatal(err)
	}

	signatureHex := hex.EncodeToString(signature)
	fmt.Printf("签名长度: %d 字节\n", len(signature))
	fmt.Printf("签名数据: 0x%s\n", signatureHex)

	return signatureHex
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Println("\n===== 以太坊密钥工具 =====")
		fmt.Println("1. 生成新的密钥对")
		fmt.Println("2. 签名数据")
		fmt.Println("3. 退出")
		fmt.Print("请输入选项 (1/2/3): ")

		choice, _ := reader.ReadString('\n')
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			fmt.Println("\n==== 生成新的密钥对 ====")
			fmt.Println(createWallet())

		case "2":
			fmt.Println("\n==== 数据签名 ====")
			// 检查是否有保存的地址
			if len(addressToPrivateKey) == 0 {
				fmt.Println("当前没有可用的地址，请先生成密钥对")
				continue
			}

			// 显示可用地址
			fmt.Println("可用地址:")
			for address := range addressToPrivateKey {
				fmt.Println("-", address)
			}

			fmt.Print("请输入要使用的地址: ")
			address, _ := reader.ReadString('\n')
			address = strings.TrimSpace(address)

			if _, exists := addressToPrivateKey[address]; !exists {
				fmt.Println("错误: 找不到此地址")
				continue
			}

			fmt.Print("请输入要签名的数据 (16进制，可带0x前缀): ")
			data, _ := reader.ReadString('\n')
			data = strings.TrimSpace(data)

			signData(data, address)

		case "3":
			fmt.Println("程序已退出")
			return

		default:
			fmt.Println("无效选项，请重新输入")
		}
	}
}
