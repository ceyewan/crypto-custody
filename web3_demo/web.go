package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

// 创建钱包
func createWallet() {
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
	fmt.Printf("X: %x\n", publicKey.(*ecdsa.PublicKey).X)
	fmt.Printf("Y: %x\n", publicKey.(*ecdsa.PublicKey).Y)
	fmt.Printf("地址: %s\n", address)
	fmt.Printf("私钥: %s\n", privateKeyStr)
}

// 获取指定地址的余额
func getBalance(address string, client *ethclient.Client) {
	// 指定要查询的地址，将字符串地址转换为以太坊地址格式
	addr := common.HexToAddress(address)

	// 从以太坊客户端获取指定地址的余额
	// context.Background()是一个空的上下文，上下文在 Go 中用于控制并发操作的生命周期，例如取消操作或传递请求范围的数据。
	// nil表示查询最新的区块
	balance, err := client.BalanceAt(context.Background(), addr, nil)
	if err != nil {
		log.Fatal(err)
	}
	// 将余额从Wei转换为Ether，Wei是以太坊中最小的单位，而Ether是以太坊的主要货币单位。1 Ether等于10^18 Wei。
	fbalance := new(big.Float)
	// SetString 方法会解析传入的字符串并将其转换为浮点数
	fbalance.SetString(balance.String())
	// Quo 方法用于执行两个大浮点数之间的除法运算，并返回结果
	ethValue := new(big.Float).Quo(fbalance, big.NewFloat(1e18))

	fmt.Printf("账户余额: %s ETH\n", ethValue.Text('f', 18))
}

// 将一个账户中的ETH转账到另一个账户
func transfer(fromAddress string, toAddress string, client *ethclient.Client) {
	// 指定发送方和接收方地址
	fromAddr := common.HexToAddress(fromAddress)
	toAddr := common.HexToAddress(toAddress)
	// 获取发送方的私钥
	privateKey, err := crypto.HexToECDSA("8ff47f75ce3e6d084c03007667ae896c85c220a0b34194801b1cc8dd40599ab9")
	if err != nil {
		log.Fatal(err)
	}
	// nonce 是一个用于标识以太坊账户交易顺序的计数器。每当账户发起一笔新交易时，nonce 值会递增。
	// PendingNonceAt 方法用于获取指定地址的下一个待处理交易的 nonce 值
	nonce, err := client.PendingNonceAt(context.Background(), fromAddr)
	if err != nil {
		log.Fatal(err)
	}
	// 设置转账金额（以Wei为单位）
	value := big.NewInt(10000000000000000) // 0.01 ETH
	// 设置gas限制，这个数值通常用于简单的以太坊转账操作
	gasLimit := uint64(21000) // 标准转账的gas限制
	// 获取当前gas价格
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	// 创建一个新的交易实例，并设置交易的各个属性
	tx := types.NewTransaction(nonce, toAddr, value, gasLimit, gasPrice, nil)

	// 获取以太坊网络的链ID
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	// SignTx 调用crypto.Sign方法生成签名，并将签名附加到交易上并返回一个已签名的交易实例。
	// NewEIP155Signer 方法用于创建一个新的 EIP-155 签名者实例，并设置链ID。
	// EIP-155 是以太坊改进提案 155，它引入了一种新的交易签名方法，通过在签名中包含 chainID 来防止跨链重放攻击。
	// chainID 是以太坊网络的链标识符，不同的以太坊网络（例如主网、测试网）有不同的 chainID。在签名交易时，chainID 被用来确保交易只能在特定的网络上有效，从而防止跨链重放攻击。
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		log.Fatal(err)
	}

	// 将签名交易发送到以太坊网络
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("交易已发送: %s\n", signedTx.Hash().Hex())
}

func packTransferData(fromAddress string, toAddress string, client *ethclient.Client, value float32) (types.EIP155Signer, *types.Transaction, string) {
	// 指定发送方和接收方地址
	fromAddr := common.HexToAddress(fromAddress)
	toAddr := common.HexToAddress(toAddress)
	// nonce 是一个用于标识以太坊账户交易顺序的计数器。每当账户发起一笔新交易时，nonce 值会递增。
	// PendingNonceAt 方法用于获取指定地址的下一个待处理交易的 nonce 值
	nonce, err := client.PendingNonceAt(context.Background(), fromAddr)
	if err != nil {
		log.Fatal(err)
	}
	// 设置转账金额（以Wei为单位）输入 value 为转账金额， ETH 为单位
	amount := new(big.Int)
	amount.SetString(fmt.Sprintf("%d", int64(value*1e18)), 10)
	// 设置gas限制，这个数值通常用于简单的以太坊转账操作
	gasLimit := uint64(21000) // 标准转账的gas限制
	// 获取当前gas价格
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	// 创建一个新的交易实例，并设置交易的各个属性
	tx := types.NewTransaction(nonce, toAddr, amount, gasLimit, gasPrice, nil)

	// 获取以太坊网络的链ID
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	s := types.NewEIP155Signer(chainID)
	fmt.Println("交易数据:", tx)
	h := s.Hash(tx)
	fmt.Println("长度:", len([]byte(h[:])), "签名数据:", []byte(h[:]))
	return s, tx, hex.EncodeToString(h[:])
}

func signedData(data string, privateKeyHex string) string {
	// 生成私钥
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		log.Fatal(err)
	}
	// data 转 common.Hash
	var hash common.Hash
	hashBytes, _ := hex.DecodeString(data)
	copy(hash[:], hashBytes)
	// 签名数据
	signature, err := crypto.Sign(hash[:], privateKey)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("长度:", len(signature), "签名数据:", signature)
	return hex.EncodeToString(signature)
}

// 使用签名数据发送交易
func sendTransfer(s types.EIP155Signer, tx *types.Transaction, sign string, client *ethclient.Client) {
	signedData, err := hex.DecodeString(sign)
	fmt.Println("长度:", len(signedData), "签名数据:", signedData)
	if err != nil {
		log.Fatal(err)
	}
	signedTx, err := tx.WithSignature(s, signedData)
	if err != nil {
		log.Fatal(err)
	}
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("交易已发送: %s\n", signedTx.Hash().Hex())
	// 等待交易确认
	receipt, err := bind.WaitMined(context.Background(), client, signedTx)
	if err != nil {
		log.Fatal(err)
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		log.Fatal("交易失败")
	}
}

func PubkeyToAddress(X string, Y string) string {
	// 生成公钥
	xInt := new(big.Int)
	yInt := new(big.Int)
	xInt.SetString(X, 16)
	yInt.SetString(Y, 16)
	pubKey := crypto.PubkeyToAddress(ecdsa.PublicKey{X: xInt, Y: yInt})
	fmt.Println("地址:", pubKey.Hex())
	return pubKey.Hex()
}

// ParseSignatureFromRSV 从 r、s、v 解析签名数据
func ParseSignatureFromRSV(r, s *big.Int, v int) []byte {
	// 创建65字节的签名
	signature := make([]byte, 65)
	// 填充 R (32字节)
	rBytes := r.Bytes()
	copy(signature[32-len(rBytes):32], rBytes)
	// 填充 S (32字节)
	sBytes := s.Bytes()
	copy(signature[64-len(sBytes):64], sBytes)
	// 设置 V (1字节)
	signature[64] = byte(v)
	return signature
}

// ParseSignatureFromString 从字符串解析签名数据
func ParseSignatureFromString(rStr, sStr, pubX, pubY, data string) (string, error) {
	fmt.Println("rStr:", rStr, "sStr:", sStr, "pubX:", pubX, "pubY:", pubY, "data:", data)
	// 将字符串转换为 big.Int
	rInt := new(big.Int)
	rInt.SetString(rStr, 16)
	sInt := new(big.Int)
	sInt.SetString(sStr, 16)
	// 将字符串 hash 转换为 common.Hash
	hash := common.HexToHash(data)
	recid := GetRecid(rInt, sInt, hash.Bytes(), pubX, pubY)
	if recid == -1 {
		log.Fatal("无法恢复公钥")
	}
	return hex.EncodeToString(ParseSignatureFromRSV(rInt, sInt, recid)), nil
}

// 使用 r、s 和公钥计算得到 recid
func GetRecid(r, s *big.Int, msg []byte, pubX, pubY string) int {
	// 分别假设 recid 为 0、1，如何和 r、s 组成签名
	// 从签名中恢复公钥，然后和给定的公钥比较
	for i := 0; i < 2; i++ {
		signature := ParseSignatureFromRSV(r, s, i)
		// 从签名中恢复公钥
		pubKey, err := crypto.Ecrecover(msg, signature)
		if err != nil {
			log.Fatal(err)
		}
		// 比较恢复的公钥和给定的公钥
		fmt.Println("公钥:", hex.EncodeToString(pubKey))
		fmt.Println("给定的公钥:", "04"+pubX+pubY)
		if strings.EqualFold(hex.EncodeToString(pubKey), "04"+pubX+pubY) {
			return i
		}
	}
	return -1
}

func main() {
	// // 连接到Sepolia测试网
	// client, err := ethclient.Dial("https://sepolia.infura.io/v3/766c230ed91a48a097e2739b966bbbf7")
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// var pubX, pubY string
	// fmt.Print("请输入公钥 X: ")
	// fmt.Scanln(&pubX)
	// fmt.Print("请输入公钥 Y: ")
	// fmt.Scanln(&pubY)
	// // createWallet()
	// address := PubkeyToAddress(pubX, pubY)

	// // getBalance("0x007d10c5222c2f326D0145Ab9B6148C6ED9c909d", client)
	// // getBalance(address, client)
	// // s, tx, data := packTransferData("0x007d10c5222c2f326D0145Ab9B6148C6ED9c909d", address, client, 0.002)
	// // fmt.Println("交易哈希的十六进制字符串表示:", data)
	// // sign := signedData(data)
	// // fmt.Println("签名数据:", sign)
	// // sendTransfer(s, tx, sign, client)
	// // getBalance("0x007d10c5222c2f326D0145Ab9B6148C6ED9c909d", client)
	// // getBalance(address, client)
	// fmt.Printf("\n====================================\n")
	// getBalance("0x007d10c5222c2f326D0145Ab9B6148C6ED9c909d", client)
	// getBalance(address, client)
	// s, tx, data := packTransferData(address, "0x007d10c5222c2f326D0145Ab9B6148C6ED9c909d", client, 0.001)
	// fmt.Println("交易哈希的十六进制字符串表示:", data)
	// // 等待用户输入 r、s 和 recid
	// var rStr, sStr string
	// fmt.Print("请输入 r: ")
	// fmt.Scanln(&rStr)
	// fmt.Print("请输入 s: ")
	// fmt.Scanln(&sStr)
	// sign, _ := ParseSignatureFromString(rStr, sStr, pubX, pubY, data)
	// fmt.Println("签名数据:", sign)
	// sendTransfer(s, tx, sign, client)
	// getBalance("0x007d10c5222c2f326D0145Ab9B6148C6ED9c909d", client)
	// getBalance(address, client)

	// rt, _ := ParseSignatureFromString("91f7366be13227aff457cf4eb9559a983a1861d26fa65cdf37f294b9f5c51601", "5578cefb69ee55905d49341b82053692fbacfc652df4c4550ff579be74a77f82", "87043c9ad380a2cdd78a3854cbe2603548a881c099e64440c2e0e6c8f1296b74", "49fe671eaf1f54497d01b1603a1277427f7963bf300bb6c6a9444c75e0006ce0", "4cca684310ba248350490ab3e82363f147e0ab7c3ff78454c1b53d04b3dab60a")
	// fmt.Println("签名数据:", rt)
	createWallet()
	fmt.Println(signedData("9ab47ded8b4c63192140e96f94dc7378a9d5717440aa2058a1244d886ef98fd4", "4d21e7794ac4ed8310945ea72c6b260ac24d6e5803fd9456217de0832f6c7bec"))
}
