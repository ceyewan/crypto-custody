package main

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

type options struct {
	onlineURL      string
	onlineUser     string
	onlinePassword string
	ganacheRPC     string
	fundAmount     string
	txValue        string
	outFile        string
	waitTimeout    time.Duration
	skipFund       bool
	showPrivateKey bool
}

type apiClient struct {
	baseURL string
	token   string
	client  *http.Client
}

type standardResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func main() {
	var opts options
	flag.StringVar(&opts.onlineURL, "online-url", envDefault("ONLINE_URL", "http://127.0.0.1:8088"), "online system HTTP URL")
	flag.StringVar(&opts.onlineUser, "online-user", envDefault("ONLINE_USER", "admin"), "online username")
	flag.StringVar(&opts.onlinePassword, "online-password", envDefault("ONLINE_PASSWORD", "admin123"), "online password")
	flag.StringVar(&opts.ganacheRPC, "ganache-rpc", envDefault("GANACHE_RPC_URL", "http://127.0.0.1:8545"), "Ganache JSON-RPC URL")
	flag.StringVar(&opts.fundAmount, "fund-amount", envDefault("GANACHE_FUND_AMOUNT", "2"), "ETH amount to fund the generated source address")
	flag.StringVar(&opts.txValue, "tx-value", envDefault("ECDSA_LOOP_TX_VALUE", "0.1 ETH"), "ETH transaction value sent through the online system")
	flag.StringVar(&opts.outFile, "out", "", "optional JSON report path")
	flag.DurationVar(&opts.waitTimeout, "wait-timeout", 20*time.Second, "receipt wait timeout")
	flag.BoolVar(&opts.skipFund, "skip-fund", false, "skip Ganache funding step")
	flag.BoolVar(&opts.showPrivateKey, "show-private-key", false, "include the generated source private key in the JSON report")
	flag.Parse()

	report, err := run(opts)
	if err != nil {
		report["success"] = false
		report["error"] = err.Error()
		_ = writeReport(opts.outFile, report)
		printReport(report)
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		os.Exit(1)
	}
	report["success"] = true
	if err := writeReport(opts.outFile, report); err != nil {
		fmt.Fprintln(os.Stderr, "FAIL:", err)
		os.Exit(1)
	}
	printReport(report)
}

func run(opts options) (map[string]any, error) {
	report := map[string]any{
		"online_url":  opts.onlineURL,
		"ganache_rpc": opts.ganacheRPC,
		"tx_value":    opts.txValue,
	}

	chainID, err := ganacheChainID(opts.ganacheRPC)
	if err != nil {
		return report, err
	}
	report["ganache_chain_id"] = chainID.String()

	sourceKey, err := ethcrypto.GenerateKey()
	if err != nil {
		return report, err
	}
	targetKey, err := ethcrypto.GenerateKey()
	if err != nil {
		return report, err
	}
	sourceAddress := ethcrypto.PubkeyToAddress(sourceKey.PublicKey).Hex()
	targetAddress := ethcrypto.PubkeyToAddress(targetKey.PublicKey).Hex()
	report["source_address"] = sourceAddress
	report["target_address"] = targetAddress
	if opts.showPrivateKey {
		report["source_private_key"] = "0x" + hex.EncodeToString(ethcrypto.FromECDSA(sourceKey))
	}

	if !opts.skipFund {
		fundResult, err := ganacheSend(opts.ganacheRPC, sourceAddress, opts.fundAmount, opts.waitTimeout)
		if err != nil {
			return report, err
		}
		report["funding"] = fundResult
	}

	sourceBefore, err := ganacheBalance(opts.ganacheRPC, sourceAddress)
	if err != nil {
		return report, err
	}
	targetBefore, err := ganacheBalance(opts.ganacheRPC, targetAddress)
	if err != nil {
		return report, err
	}
	report["balances_before"] = balanceReport(sourceAddress, sourceBefore, targetAddress, targetBefore)

	online := &apiClient{
		baseURL: strings.TrimRight(opts.onlineURL, "/"),
		client:  &http.Client{Timeout: 30 * time.Second},
	}
	loginRaw, err := online.postRaw("/api/login", map[string]any{
		"identifier": opts.onlineUser,
		"username":   opts.onlineUser,
		"password":   opts.onlinePassword,
	})
	if err != nil {
		return report, fmt.Errorf("online login: %w", err)
	}
	loginData, err := responseDataMap(loginRaw)
	if err != nil {
		return report, fmt.Errorf("online login response: %w", err)
	}
	online.token = stringField(loginData, "token")
	if online.token == "" {
		return report, fmt.Errorf("online login returned empty token")
	}
	report["online_user"] = opts.onlineUser

	createRaw, err := online.postRaw("/api/transactions", map[string]any{
		"txType":      "withdraw",
		"fromAddress": sourceAddress,
		"toAddress":   targetAddress,
		"value":       opts.txValue,
		"coinType":    "ETH",
		"reason":      "minimal Go ECDSA signing loop",
	})
	if err != nil {
		return report, fmt.Errorf("online create transaction: %w", err)
	}
	createData, err := responseDataMap(createRaw)
	if err != nil {
		return report, fmt.Errorf("online create response: %w", err)
	}
	txID := uint(numberField(createData, "ID", "id"))
	messageHash := stringField(createData, "MessageHash", "messageHash", "message_hash")
	if txID == 0 {
		return report, fmt.Errorf("online create response missing transaction ID: %s", string(createRaw))
	}
	if messageHash == "" {
		return report, fmt.Errorf("online create response missing message hash: %s", string(createRaw))
	}
	report["online_transaction_id"] = txID
	report["message_hash"] = messageHash

	signature, verifyReport, err := signAndVerify(messageHash, sourceKey)
	report["local_signature_check"] = verifyReport
	if err != nil {
		return report, err
	}
	report["signature"] = "0x" + hex.EncodeToString(signature)

	importRaw, err := online.postRaw(fmt.Sprintf("/api/transactions/%d/import-signature", txID), map[string]any{
		"messageHash": messageHash,
		"signature":   "0x" + hex.EncodeToString(signature),
		"completedAt": time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return report, fmt.Errorf("online import signature: %w", err)
	}
	importData, err := responseDataMap(importRaw)
	if err != nil {
		return report, fmt.Errorf("online import response: %w", err)
	}
	report["online_import_status"] = stringField(importData, "Status", "status")

	broadcastRaw, err := online.postRaw(fmt.Sprintf("/api/transactions/%d/broadcast", txID), nil)
	if err != nil {
		return report, fmt.Errorf("online broadcast transaction: %w", err)
	}
	broadcastData, err := responseDataMap(broadcastRaw)
	if err != nil {
		return report, fmt.Errorf("online broadcast response: %w", err)
	}
	txHash := stringField(broadcastData, "TxHash", "txHash", "tx_hash")
	if txHash == "" {
		return report, fmt.Errorf("online broadcast response missing tx hash: %s", string(broadcastRaw))
	}
	report["broadcast_tx_hash"] = txHash

	receipt, err := waitForReceipt(opts.ganacheRPC, txHash, opts.waitTimeout)
	if err != nil {
		return report, err
	}
	report["receipt"] = receipt
	if status := stringField(receipt, "status"); status != "" && status != "0x1" {
		return report, fmt.Errorf("broadcast receipt status is %s", status)
	}

	sourceAfter, err := ganacheBalance(opts.ganacheRPC, sourceAddress)
	if err != nil {
		return report, err
	}
	targetAfter, err := ganacheBalance(opts.ganacheRPC, targetAddress)
	if err != nil {
		return report, err
	}
	report["balances_after"] = balanceReport(sourceAddress, sourceAfter, targetAddress, targetAfter)

	txValueWei, err := ethToWei(opts.txValue)
	if err != nil {
		return report, err
	}
	targetDelta := new(big.Int).Sub(targetAfter, targetBefore)
	report["target_delta_wei"] = targetDelta.String()
	report["target_delta_eth"] = weiToETH(targetDelta)
	if targetDelta.Cmp(txValueWei) != 0 {
		return report, fmt.Errorf("target balance delta=%s wei, want tx value=%s wei", targetDelta.String(), txValueWei.String())
	}

	sourceDelta := new(big.Int).Sub(sourceBefore, sourceAfter)
	report["source_spent_wei"] = sourceDelta.String()
	report["source_spent_eth"] = weiToETH(sourceDelta)
	if sourceDelta.Cmp(txValueWei) < 0 {
		return report, fmt.Errorf("source spent=%s wei, expected at least tx value %s wei", sourceDelta.String(), txValueWei.String())
	}

	return report, nil
}

func signAndVerify(messageHash string, privateKey *ecdsa.PrivateKey) ([]byte, map[string]any, error) {
	report := map[string]any{
		"message_hash": messageHash,
	}
	hashBytes, err := decodeFixedHex(messageHash, 32, "message_hash")
	if err != nil {
		return nil, report, err
	}
	signature, err := ethcrypto.Sign(hashBytes, privateKey)
	if err != nil {
		return nil, report, err
	}
	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:64])
	report["signature"] = "0x" + hex.EncodeToString(signature)
	report["r"] = r.String()
	report["s"] = s.String()
	report["v"] = int(signature[64])
	report["low_s_valid"] = ethcrypto.ValidateSignatureValues(signature[64], r, s, true)

	pubKey, err := ethcrypto.SigToPub(hashBytes, signature)
	if err != nil {
		return nil, report, fmt.Errorf("local SigToPub: %w", err)
	}
	recovered := ethcrypto.PubkeyToAddress(*pubKey).Hex()
	expected := ethcrypto.PubkeyToAddress(privateKey.PublicKey).Hex()
	report["recovered_address"] = recovered
	report["expected_address"] = expected
	if !strings.EqualFold(recovered, expected) {
		return nil, report, fmt.Errorf("local signature recovered %s, want %s", recovered, expected)
	}
	return signature, report, nil
}

func (c *apiClient) postRaw(path string, body any) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", c.token)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(raw))
	}
	return raw, nil
}

func responseDataMap(raw []byte) (map[string]any, error) {
	var resp standardResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	if resp.Code >= 400 {
		return nil, fmt.Errorf("online API error %d: %s", resp.Code, resp.Message)
	}
	if len(resp.Data) == 0 {
		return map[string]any{}, nil
	}
	var data map[string]any
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func ganacheChainID(rpcURL string) (*big.Int, error) {
	var chainIDHex string
	if err := rpcCall(rpcURL, "eth_chainId", []any{}, &chainIDHex); err != nil {
		return nil, fmt.Errorf("Ganache eth_chainId: %w", err)
	}
	chainID := new(big.Int)
	if _, ok := chainID.SetString(strings.TrimPrefix(chainIDHex, "0x"), 16); !ok {
		return nil, fmt.Errorf("invalid Ganache chain id: %s", chainIDHex)
	}
	return chainID, nil
}

func ganacheAccounts(rpcURL string) ([]string, error) {
	var accounts []string
	if err := rpcCall(rpcURL, "eth_accounts", []any{}, &accounts); err != nil {
		return nil, fmt.Errorf("Ganache eth_accounts: %w", err)
	}
	return accounts, nil
}

func ganacheSend(rpcURL, toAddress, amountETH string, waitTimeout time.Duration) (map[string]any, error) {
	accounts, err := ganacheAccounts(rpcURL)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, fmt.Errorf("Ganache returned no unlocked accounts")
	}
	value, err := ethToWei(amountETH)
	if err != nil {
		return nil, err
	}
	args := map[string]any{
		"from":  accounts[0],
		"to":    toAddress,
		"value": "0x" + value.Text(16),
	}
	var txHash string
	if err := rpcCall(rpcURL, "eth_sendTransaction", []any{args}, &txHash); err != nil {
		return nil, err
	}
	receipt, err := waitForReceipt(rpcURL, txHash, waitTimeout)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"from":       accounts[0],
		"to":         toAddress,
		"amount_eth": amountETH,
		"amount_wei": value.String(),
		"tx_hash":    txHash,
		"receipt":    receipt,
	}, nil
}

func ganacheBalance(rpcURL, address string) (*big.Int, error) {
	var hexBalance string
	if err := rpcCall(rpcURL, "eth_getBalance", []any{address, "latest"}, &hexBalance); err != nil {
		return nil, fmt.Errorf("Ganache eth_getBalance %s: %w", address, err)
	}
	value := new(big.Int)
	if _, ok := value.SetString(strings.TrimPrefix(hexBalance, "0x"), 16); !ok {
		return nil, fmt.Errorf("invalid balance hex for %s: %s", address, hexBalance)
	}
	return value, nil
}

func waitForReceipt(rpcURL, txHash string, timeout time.Duration) (map[string]any, error) {
	deadline := time.Now().Add(timeout)
	for {
		var receipt map[string]any
		if err := rpcCall(rpcURL, "eth_getTransactionReceipt", []any{txHash}, &receipt); err != nil {
			return nil, fmt.Errorf("Ganache receipt %s: %w", txHash, err)
		}
		if receipt != nil {
			return receipt, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for receipt %s", txHash)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func rpcCall(rpcURL, method string, params []any, out any) error {
	payload := map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params}
	raw, _ := json.Marshal(payload)
	resp, err := http.Post(rpcURL, "application/json", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}
	var decoded struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return err
	}
	if decoded.Error != nil {
		return fmt.Errorf("RPC %d: %s", decoded.Error.Code, decoded.Error.Message)
	}
	if string(decoded.Result) == "null" {
		return nil
	}
	return json.Unmarshal(decoded.Result, out)
}

func ethToWei(text string) (*big.Int, error) {
	cleaned := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(text), "ETH"))
	cleaned = strings.TrimSpace(strings.TrimSuffix(cleaned, "eth"))
	value := new(big.Float)
	if _, ok := value.SetString(cleaned); !ok {
		return nil, fmt.Errorf("invalid ETH amount: %s", text)
	}
	weiFloat := new(big.Float).Mul(value, big.NewFloat(1e18))
	wei := new(big.Int)
	weiFloat.Int(wei)
	return wei, nil
}

func weiToETH(wei *big.Int) string {
	if wei == nil {
		return "0"
	}
	value := new(big.Float).SetInt(wei)
	eth := new(big.Float).Quo(value, big.NewFloat(1e18))
	return eth.Text('f', 18)
}

func balanceReport(sourceAddress string, sourceBalance *big.Int, targetAddress string, targetBalance *big.Int) map[string]any {
	return map[string]any{
		sourceAddress: map[string]string{
			"wei": sourceBalance.String(),
			"eth": weiToETH(sourceBalance),
		},
		targetAddress: map[string]string{
			"wei": targetBalance.String(),
			"eth": weiToETH(targetBalance),
		},
	}
}

func decodeFixedHex(value string, length int, label string) ([]byte, error) {
	cleaned := strings.TrimPrefix(strings.TrimSpace(value), "0x")
	decoded, err := hex.DecodeString(cleaned)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", label, err)
	}
	if len(decoded) != length {
		return nil, fmt.Errorf("%s length=%d, want %d bytes", label, len(decoded), length)
	}
	return decoded, nil
}

func stringField(data map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := data[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case string:
			return typed
		case fmt.Stringer:
			return typed.String()
		case float64:
			return strconv.FormatFloat(typed, 'f', -1, 64)
		case json.Number:
			return typed.String()
		}
	}
	return ""
}

func numberField(data map[string]any, keys ...string) float64 {
	for _, key := range keys {
		value, ok := data[key]
		if !ok {
			continue
		}
		switch typed := value.(type) {
		case float64:
			return typed
		case json.Number:
			parsed, _ := typed.Float64()
			return parsed
		case string:
			parsed, _ := strconv.ParseFloat(typed, 64)
			return parsed
		}
	}
	return 0
}

func envDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func writeReport(path string, report map[string]any) error {
	if path == "" {
		return nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(abs, append(raw, '\n'), 0644)
}

func printReport(report map[string]any) {
	raw, _ := json.MarshalIndent(report, "", "  ")
	fmt.Println(string(raw))
}
