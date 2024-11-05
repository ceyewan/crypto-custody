package main

import (
	"context"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func getBalance(address string, client *ethclient.Client) {
	// 指定要查询的地址
	addr := common.HexToAddress(address)

	// 获取余额
	balance, err := client.BalanceAt(context.Background(), addr, nil)
	if err != nil {
		log.Fatal(err)
	}

	// 将余额从Wei转换为Ether
	fbalance := new(big.Float)
	fbalance.SetString(balance.String())
	ethValue := new(big.Float).Quo(fbalance, big.NewFloat(1e18))

	fmt.Printf("账户余额: %s ETH\n", ethValue.Text('f', 18))
}

func main() {
	// 连接到Sepolia测试网
	client, err := ethclient.Dial("https://sepolia.infura.io/v3/766c230ed91a48a097e2739b966bbbf7")
	if err != nil {
		log.Fatal(err)
	}

	// 指定要查询的地址
	address := common.HexToAddress("0x007d10c5222c2f326D0145Ab9B6148C6ED9c909d")

	// 获取余额
	balance, err := client.BalanceAt(context.Background(), address, nil)
	if err != nil {
		log.Fatal(err)
	}

	// 将余额从Wei转换为Ether
	fbalance := new(big.Float)
	fbalance.SetString(balance.String())
	ethValue := new(big.Float).Quo(fbalance, big.NewFloat(1e18))

	fmt.Printf("账户余额: %s ETH\n", ethValue.Text('f', 18))
}
