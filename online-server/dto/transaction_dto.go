package dto

// 准备交易请求
type PrepareTransactionRequest struct {
	FromAddress string  `json:"fromAddress" binding:"required"`
	ToAddress   string  `json:"toAddress" binding:"required"`
	Amount      float64 `json:"amount" binding:"required,gt=0"`
}

// 签名并发送交易请求
type SignSendTransactionRequest struct {
	MessageHash string `json:"messageHash" binding:"required"`
	Signature   string `json:"signature" binding:"required"`
}

// 余额响应
type BalanceResponse struct {
	Address string `json:"address"`
	Balance string `json:"balance"`
}

// 预备交易响应
type PrepareTransactionResponse struct {
	TransactionID uint   `json:"transactionId"`
	MessageHash   string `json:"messageHash"`
}

// 交易发送响应
type SendTransactionResponse struct {
	TransactionID uint   `json:"transactionId"`
	TxHash        string `json:"txHash"`
}
