package servers

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var tx *types.Transaction
var s types.EIP155Signer

// 将一个账户中的ETH转账到另一个账户
func Transfer(toAddress string) {
	client, err := ethclient.Dial("https://sepolia.infura.io/v3/766c230ed91a48a097e2739b966bbbf7")
	if err != nil {
		log.Fatal(err)
	}
	// 指定发送方和接收方地址
	fromAddr := common.HexToAddress("0x007d10c5222c2f326D0145Ab9B6148C6ED9c909d")
	toAddr := common.HexToAddress(toAddress)
	privateKey, err := crypto.HexToECDSA("8ff47f75ce3e6d084c03007667ae896c85c220a0b34194801b1cc8dd40599ab9")
	if err != nil {
		log.Fatal(err)
	}
	nonce, err := client.PendingNonceAt(context.Background(), fromAddr)
	if err != nil {
		log.Fatal(err)
	}
	value := big.NewInt(10000000000000000) // 0.01 ETH
	gasLimit := uint64(21000)              // 标准转账的gas限制
	// 获取当前gas价格
	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	tx := types.NewTransaction(nonce, toAddr, value, gasLimit, gasPrice, nil)
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		log.Fatal(err)
	}
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("交易已发送: %s\n", signedTx.Hash().Hex())
}

// 获取指定地址的余额
func GetBalance(address string) *big.Float {
	client, err := ethclient.Dial("https://sepolia.infura.io/v3/766c230ed91a48a097e2739b966bbbf7")
	if err != nil {
		log.Fatal(err)
	}
	addr := common.HexToAddress(address)
	balance, err := client.BalanceAt(context.Background(), addr, nil)
	if err != nil {
		log.Fatal(err)
	}
	fbalance := new(big.Float)
	fbalance.SetString(balance.String())
	ethValue := new(big.Float).Quo(fbalance, big.NewFloat(1e18))
	return ethValue
}

func PackTransferData(fromAddress string, toAddress string, value float64) string {
	client, err := ethclient.Dial("https://sepolia.infura.io/v3/766c230ed91a48a097e2739b966bbbf7")
	if err != nil {
		log.Fatal(err)
	}
	// 指定发送方和接收方地址
	fromAddr := common.HexToAddress(fromAddress)
	toAddr := common.HexToAddress(toAddress)
	nonce, err := client.PendingNonceAt(context.Background(), fromAddr)
	if err != nil {
		log.Fatal(err)
	}
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
	tx = types.NewTransaction(nonce, toAddr, amount, gasLimit, gasPrice, nil)
	// 获取以太坊网络的链ID
	chainID, err := client.NetworkID(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	s = types.NewEIP155Signer(chainID)
	fmt.Println("交易数据:", tx)
	h := s.Hash(tx)
	fmt.Println("长度:", len([]byte(h[:])), "签名数据:", []byte(h[:]))
	return hex.EncodeToString(h[:])
}

// 使用签名数据发送交易
func SendTransfer(sign string) error {
	client, err := ethclient.Dial("https://sepolia.infura.io/v3/766c230ed91a48a097e2739b966bbbf7")
	if err != nil {
		return err
	}
	fmt.Println("交易数据:", tx)
	signedData, err := hex.DecodeString(sign)
	fmt.Println("长度:", len(signedData), "签名数据:", signedData)
	if err != nil {
		return err
	}
	signedTx, err := tx.WithSignature(s, signedData)
	if err != nil {
		return err
	}
	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return err
	}
	fmt.Printf("交易已发送: %s\n", signedTx.Hash().Hex())
	// 等待交易确认
	receipt, err := bind.WaitMined(context.Background(), client, signedTx)
	if err != nil {
		return err
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("交易失败: %d", receipt.Status)
	}
	return nil
}
