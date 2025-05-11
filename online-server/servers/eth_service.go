package servers

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

var (
	// EthClient is a global client to interact with Ethereum networks
	EthClient *ethclient.Client

	// CurrentNonce stores the current nonce for the transaction
	CurrentNonce uint64

	// TransferData holds data for the current transaction being processed
	TransferData string

	// RpcEndpoint is the default Ethereum RPC endpoint
	RpcEndpoint = "https://sepolia.infura.io/v3/766c230ed91a48a097e2739b966bbbf7"

	// ChainID represents the Ethereum network ID (Sepolia = 11155111)
	ChainID = big.NewInt(11155111)
)

// InitEthService initializes the Ethereum service
func InitEthService() error {
	var err error
	EthClient, err = ethclient.Dial(RpcEndpoint)
	if err != nil {
		return fmt.Errorf("failed to connect to the Ethereum client: %v", err)
	}
	return nil
}

// GetBalance retrieves the balance of an Ethereum address
func GetBalance(address string) *big.Float {
	account := common.HexToAddress(address)
	balance, err := EthClient.BalanceAt(nil, account, nil)
	if err != nil {
		return new(big.Float)
	}
	
	// Convert Wei to Ether
	fbalance := new(big.Float)
	fbalance.SetString(balance.String())
	ethValue := new(big.Float).Quo(fbalance, big.NewFloat(1e18))
	return ethValue
}

// Transfer transfers ETH to a specified address
func Transfer(to string) error {
	// Implementation for transferring ETH
	// This is a simplified version, in a real implementation,
	// you would need to handle private keys securely
	return nil
}

// PackTransferData prepares transaction data for a transfer
func PackTransferData(from, to string, amount float64) string {
	// Get nonce
	fromAddress := common.HexToAddress(from)
	nonce, err := EthClient.PendingNonceAt(nil, fromAddress)
	if err != nil {
		return ""
	}
	CurrentNonce = nonce

	// Convert ETH amount to Wei
	value := new(big.Int)
	ethAmount := new(big.Float)
	ethAmount.SetFloat64(amount)
	
	// 1 ETH = 10^18 Wei
	weiAmount := new(big.Float).Mul(ethAmount, big.NewFloat(1e18))
	weiAmount.Int(value)

	// Gas settings
	gasLimit := uint64(21000) // Standard ETH transfer
	gasPrice, err := EthClient.SuggestGasPrice(nil)
	if err != nil {
		return ""
	}

	// Prepare transaction
	tx := types.NewTransaction(
		nonce,
		common.HexToAddress(to),
		value,
		gasLimit,
		gasPrice,
		nil,
	)

	// Get the signing hash
	signer := types.NewEIP155Signer(ChainID)
	hash := signer.Hash(tx)
	
	// Store data for future use and return hash
	TransferData = hex.EncodeToString(hash[:])
	return TransferData
}

// SendTransfer sends a previously prepared transaction using the signature
func SendTransfer(signature string) error {
	if TransferData == "" {
		return errors.New("no transaction data available, call PackTransferData first")
	}
	
	// Recover transaction data
	dataBytes, err := hex.DecodeString(TransferData)
	if err != nil {
		return err
	}
	
	// Convert the signature
	sig, err := hexutil.Decode("0x" + signature)
	if err != nil {
		return err
	}
	
	// Recover the public key that signed the message
	_, err = crypto.Ecrecover(dataBytes, sig)
	if err != nil {
		return err
	}
	
	// In a real implementation, you would:
	// 1. Recreate the transaction with stored parameters
	// 2. Apply the signature
	// 3. Send the transaction
	
	// Create a simple transaction (this would need to be implemented properly)
	// For demonstration purposes, we'll return an error since the real implementation
	// would require private key access
	return errors.New("transaction creation requires private key access - implement this in your client")
}