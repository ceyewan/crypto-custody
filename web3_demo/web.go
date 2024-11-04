package web3_demo

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	USDTContractAddress = "0xdac17f958d2ee523a2206206994597c13d831ec7"
	infuraURL           = "https://sepolia.infura.io/v3/your_infura_project_id"
)

func createWallet() (*ecdsa.PrivateKey, common.Address) {
	// 创建一个新的私钥
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		log.Fatal(err)
	}
	// 从私钥中获取公钥
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}
	// 从公钥中获取地址
	address := crypto.PubkeyToAddress(*publicKeyECDSA)
	return privateKey, address
}

func getBalance(client *ethclient.Client, address common.Address) (*big.Int, error) {
	balance, err := client.BalanceAt(context.Background(), address, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	// 连接到以太坊客户端
	client, err := ethclient.Dial(infuraURL)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}
	// 创建钱包
	privateKey, address := createWallet()
	// 查询余额
	balance, err := getBalance(client, address)
	if err != nil {
		log.Fatalf("Failed to get balance: %v", err)
	}
	fmt.Printf("Balance: %s USDT\n", balance.String())
}
