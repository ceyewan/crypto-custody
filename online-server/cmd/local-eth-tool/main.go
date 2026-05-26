package main

import (
	"bufio"
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const defaultRPC = "http://127.0.0.1:7545"

type txDraft struct {
	rpcURL    string
	chainID   *big.Int
	networkID *big.Int
	from      common.Address
	to        common.Address
	amount    string
	value     *big.Int
	nonce     uint64
	gasLimit  uint64
	gasPrice  *big.Int
	tx        *types.Transaction
	signer    types.Signer
	hash      common.Hash
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("本地以太坊测试工具")
	fmt.Println("1. 生成新的以太坊公私钥/地址")
	fmt.Println("2. 从已有账户私钥直接转 ETH")
	fmt.Println("3. 根据交易明细生成待签名 Hash，粘贴外部签名后广播")
	fmt.Println("4. 输入私钥和 Hash，输出标准以太坊签名")
	fmt.Print("请选择操作 [1/2/3/4]: ")

	choice := readLine(reader)
	switch choice {
	case "1":
		if err := generateAccount(); err != nil {
			log.Fatal(err)
		}
	case "2":
		if err := transferWithPrivateKey(reader); err != nil {
			log.Fatal(err)
		}
	case "3":
		if err := transferWithExternalSignature(reader); err != nil {
			log.Fatal(err)
		}
	case "4":
		if err := signHashWithPrivateKey(reader); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("未知操作: %s", choice)
	}
}

func generateAccount() error {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return fmt.Errorf("生成私钥失败: %w", err)
	}

	privateKeyBytes := crypto.FromECDSA(privateKey)
	publicKeyBytes := crypto.FromECDSAPub(&privateKey.PublicKey)
	address := crypto.PubkeyToAddress(privateKey.PublicKey)

	fmt.Println()
	fmt.Println("生成成功")
	fmt.Printf("地址: %s\n", address.Hex())
	fmt.Printf("私钥: 0x%s\n", hex.EncodeToString(privateKeyBytes))
	fmt.Printf("公钥: 0x%s\n", hex.EncodeToString(publicKeyBytes))
	fmt.Println()
	fmt.Println("说明: 转账和签名主要使用私钥；链上收款使用地址。请妥善保管私钥。")
	return nil
}

func transferWithPrivateKey(reader *bufio.Reader) error {
	draft, client, err := collectDraft(reader)
	if err != nil {
		return err
	}
	defer client.Close()

	fmt.Print("转出方私钥: ")
	privateKeyText := strings.TrimPrefix(readLine(reader), "0x")
	privateKey, err := crypto.HexToECDSA(privateKeyText)
	if err != nil {
		return fmt.Errorf("私钥格式不正确: %w", err)
	}

	derivedAddress := crypto.PubkeyToAddress(privateKey.PublicKey)
	if !strings.EqualFold(derivedAddress.Hex(), draft.from.Hex()) {
		return fmt.Errorf("私钥对应地址为 %s，与输入的转出方地址 %s 不一致", derivedAddress.Hex(), draft.from.Hex())
	}

	signedTx, err := types.SignTx(draft.tx, draft.signer, privateKey)
	if err != nil {
		return fmt.Errorf("交易签名失败: %w", err)
	}

	return sendSignedTransaction(client, draft, signedTx)
}

func transferWithExternalSignature(reader *bufio.Reader) error {
	draft, client, err := collectDraft(reader)
	if err != nil {
		return err
	}
	defer client.Close()

	printDraft(draft)
	fmt.Println()
	fmt.Println("请用离线端/门限签名系统对上面的待签名 Hash 进行签名。")
	fmt.Println("签名格式应为 65 字节十六进制: r(32) + s(32) + v(1)，可带或不带 0x 前缀。")
	fmt.Print("请输入外部签名: ")
	signatureText := strings.TrimPrefix(readLine(reader), "0x")
	signatureBytes, err := hex.DecodeString(signatureText)
	if err != nil {
		return fmt.Errorf("签名不是合法十六进制: %w", err)
	}
	if len(signatureBytes) != 65 {
		return fmt.Errorf("签名长度错误: 需要 65 字节，实际 %d 字节", len(signatureBytes))
	}
	if signatureBytes[64] >= 27 {
		signatureBytes[64] -= 27
	}

	signedTx, err := draft.tx.WithSignature(draft.signer, signatureBytes)
	if err != nil {
		return fmt.Errorf("附加签名失败: %w", err)
	}

	sender, err := types.Sender(draft.signer, signedTx)
	if err != nil {
		return fmt.Errorf("验证签名失败: %w", err)
	}
	if !strings.EqualFold(sender.Hex(), draft.from.Hex()) {
		return fmt.Errorf("签名对应地址为 %s，与转出方地址 %s 不一致", sender.Hex(), draft.from.Hex())
	}

	return sendSignedTransaction(client, draft, signedTx)
}

func signHashWithPrivateKey(reader *bufio.Reader) error {
	fmt.Print("签名私钥: ")
	privateKeyText := strings.TrimPrefix(readLine(reader), "0x")
	privateKey, err := crypto.HexToECDSA(privateKeyText)
	if err != nil {
		return fmt.Errorf("私钥格式不正确: %w", err)
	}

	address := crypto.PubkeyToAddress(privateKey.PublicKey)
	fmt.Printf("私钥对应地址: %s\n", address.Hex())

	fmt.Print("待签名 Hash: ")
	hashText := strings.TrimPrefix(readLine(reader), "0x")
	hashBytes, err := hex.DecodeString(hashText)
	if err != nil {
		return fmt.Errorf("Hash 不是合法十六进制: %w", err)
	}
	if len(hashBytes) != 32 {
		return fmt.Errorf("Hash 长度错误: 需要 32 字节，实际 %d 字节", len(hashBytes))
	}

	signature, err := crypto.Sign(hashBytes, privateKey)
	if err != nil {
		return fmt.Errorf("签名失败: %w", err)
	}

	signatureWithRecoveryID := hex.EncodeToString(signature)
	signatureWithEthereumV := make([]byte, len(signature))
	copy(signatureWithEthereumV, signature)
	signatureWithEthereumV[64] += 27

	fmt.Println()
	fmt.Println("签名完成")
	fmt.Printf("签名地址: %s\n", address.Hex())
	fmt.Printf("签名 v=0/1: 0x%s\n", signatureWithRecoveryID)
	fmt.Printf("签名 v=27/28: 0x%s\n", hex.EncodeToString(signatureWithEthereumV))
	fmt.Println()
	fmt.Println("说明: 工具第 3 个功能可直接粘贴任意一种格式，程序会自动适配 v 值。")
	return nil
}

func collectDraft(reader *bufio.Reader) (*txDraft, *ethclient.Client, error) {
	fmt.Printf("RPC 地址 [%s]: ", defaultRPC)
	rpcURL := readLine(reader)
	if rpcURL == "" {
		rpcURL = defaultRPC
	}

	fmt.Print("转出方地址: ")
	fromAddressText := readLine(reader)
	if !common.IsHexAddress(fromAddressText) {
		return nil, nil, fmt.Errorf("转出方地址格式不正确")
	}
	fromAddress := common.HexToAddress(fromAddressText)

	fmt.Print("转入方地址: ")
	toAddressText := readLine(reader)
	if !common.IsHexAddress(toAddressText) {
		return nil, nil, fmt.Errorf("转入方地址格式不正确")
	}
	toAddress := common.HexToAddress(toAddressText)

	fmt.Print("转账金额 ETH，例如 10 或 0.1: ")
	amountText := readLine(reader)
	value, err := ethToWei(amountText)
	if err != nil {
		return nil, nil, err
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, nil, fmt.Errorf("连接 RPC 失败: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	chainID, err := client.ChainID(ctx)
	if err != nil {
		client.Close()
		return nil, nil, fmt.Errorf("获取 Chain ID 失败: %w", err)
	}

	networkID, err := client.NetworkID(ctx)
	if err != nil {
		client.Close()
		return nil, nil, fmt.Errorf("获取 Network ID 失败: %w", err)
	}

	nonce, err := client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		client.Close()
		return nil, nil, fmt.Errorf("获取 nonce 失败: %w", err)
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		client.Close()
		return nil, nil, fmt.Errorf("获取 gas price 失败: %w", err)
	}

	balance, err := client.BalanceAt(ctx, fromAddress, nil)
	if err != nil {
		client.Close()
		return nil, nil, fmt.Errorf("查询转出方余额失败: %w", err)
	}

	gasLimit := uint64(21000)
	fee := new(big.Int).Mul(gasPrice, new(big.Int).SetUint64(gasLimit))
	totalNeed := new(big.Int).Add(value, fee)
	if balance.Cmp(totalNeed) < 0 {
		client.Close()
		return nil, nil, fmt.Errorf("余额不足: 当前余额 %s ETH，需要至少 %s ETH", weiToETH(balance), weiToETH(totalNeed))
	}

	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, nil)
	signer := types.NewEIP155Signer(chainID)
	hash := signer.Hash(tx)

	return &txDraft{
		rpcURL:    rpcURL,
		chainID:   chainID,
		networkID: networkID,
		from:      fromAddress,
		to:        toAddress,
		amount:    amountText,
		value:     value,
		nonce:     nonce,
		gasLimit:  gasLimit,
		gasPrice:  gasPrice,
		tx:        tx,
		signer:    signer,
		hash:      hash,
	}, client, nil
}

func sendSignedTransaction(client *ethclient.Client, draft *txDraft, signedTx *types.Transaction) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := client.SendTransaction(ctx, signedTx); err != nil {
		return fmt.Errorf("发送交易失败: %w", err)
	}

	fmt.Println()
	fmt.Println("转账已发送")
	fmt.Printf("RPC: %s\n", draft.rpcURL)
	fmt.Printf("Chain ID: %s\n", draft.chainID.String())
	fmt.Printf("Network ID: %s\n", draft.networkID.String())
	fmt.Printf("From: %s\n", draft.from.Hex())
	fmt.Printf("To: %s\n", draft.to.Hex())
	fmt.Printf("Amount: %s ETH\n", draft.amount)
	fmt.Printf("Nonce: %d\n", draft.nonce)
	fmt.Printf("Gas Limit: %d\n", draft.gasLimit)
	fmt.Printf("Gas Price: %s wei\n", draft.gasPrice.String())
	fmt.Printf("Message Hash: 0x%s\n", draft.hash.Hex()[2:])
	fmt.Printf("Tx Hash: %s\n", signedTx.Hash().Hex())
	return nil
}

func printDraft(draft *txDraft) {
	fmt.Println()
	fmt.Println("待签名交易明细")
	fmt.Printf("RPC: %s\n", draft.rpcURL)
	fmt.Printf("Chain ID: %s\n", draft.chainID.String())
	fmt.Printf("Network ID: %s\n", draft.networkID.String())
	fmt.Printf("From: %s\n", draft.from.Hex())
	fmt.Printf("To: %s\n", draft.to.Hex())
	fmt.Printf("Amount: %s ETH\n", draft.amount)
	fmt.Printf("Nonce: %d\n", draft.nonce)
	fmt.Printf("Gas Limit: %d\n", draft.gasLimit)
	fmt.Printf("Gas Price: %s wei\n", draft.gasPrice.String())
	fmt.Printf("Message Hash: 0x%s\n", draft.hash.Hex()[2:])
}

func readLine(reader *bufio.Reader) string {
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func ethToWei(amount string) (*big.Int, error) {
	if amount == "" {
		return nil, fmt.Errorf("转账金额不能为空")
	}

	f, ok := new(big.Float).SetString(amount)
	if !ok {
		return nil, fmt.Errorf("转账金额格式不正确")
	}
	if f.Sign() <= 0 {
		return nil, fmt.Errorf("转账金额必须大于 0")
	}

	weiFloat := new(big.Float).Mul(f, big.NewFloat(1e18))
	wei := new(big.Int)
	weiFloat.Int(wei)
	if wei.Sign() <= 0 {
		return nil, fmt.Errorf("转账金额过小")
	}
	return wei, nil
}

func weiToETH(wei *big.Int) string {
	f := new(big.Float).SetInt(wei)
	eth := new(big.Float).Quo(f, big.NewFloat(1e18))
	return eth.Text('f', 18)
}
