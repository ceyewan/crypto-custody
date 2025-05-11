package dto

import (
	"time"
)

// TransactionRequest 交易请求DTO
type TransactionRequest struct {
	FromAddress string  `json:"fromAddress" binding:"required"`
	ToAddress   string  `json:"toAddress" binding:"required"`
	Amount      float64 `json:"amount" binding:"required,gt=0"`
}

// TransactionResponse 交易响应DTO
type TransactionResponse struct {
	ID            uint      `json:"id"`
	FromAddress   string    `json:"fromAddress"`
	ToAddress     string    `json:"toAddress"`
	Amount        string    `json:"amount"`
	Status        string    `json:"status"`
	TxHash        string    `json:"txHash,omitempty"`
	MessageHash   string    `json:"messageHash,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	ErrorMessage  string    `json:"errorMessage,omitempty"`
	BlockNumber   uint64    `json:"blockNumber,omitempty"`
	Confirmations uint64    `json:"confirmations,omitempty"`
}

// SignatureRequest 签名请求DTO
type SignatureRequest struct {
	MessageHash string `json:"messageHash" binding:"required"`
	Signature   string `json:"signature" binding:"required"`
}

// BalanceResponse 余额响应DTO
type BalanceResponse struct {
	Address string `json:"address"`
	Balance string `json:"balance"`
	Symbol  string `json:"symbol"`
}
