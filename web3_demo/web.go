package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"

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

func main() {
	// 连接到Sepolia测试网
	// client, err := ethclient.Dial("https://sepolia.infura.io/v3/766c230ed91a48a097e2739b966bbbf7")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	createWallet()

	// getBalance("0x007d10c5222c2f326D0145Ab9B6148C6ED9c909d", client)
	// getBalance("0x274d22723e539a11C4cD294BC04c9D937eb29F4C", client)
	// transfer("0x007d10c5222c2f326D0145Ab9B6148C6ED9c909d", "0x274d22723e539a11C4cD294BC04c9D937eb29F4C", client)
}
