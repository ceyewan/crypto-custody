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

// 交易列表查询参数
type TransactionListRequest struct {
	Page     int    `form:"page" json:"page"`
	PageSize int    `form:"pageSize" json:"pageSize"`
	Status   string `form:"status" json:"status"`
	Address  string `form:"address" json:"address"`
}

// 交易详情响应
type TransactionDetailResponse struct {
	ID          uint   `json:"id"`
	FromAddress string `json:"fromAddress"`
	ToAddress   string `json:"toAddress"`
	Value       string `json:"value"`
	MessageHash string `json:"messageHash"`
	TxHash      string `json:"txHash"`
	Status      int    `json:"status"`
	StatusText  string `json:"statusText"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// 交易列表响应
type TransactionListResponse struct {
	Transactions []TransactionDetailResponse `json:"transactions"`
	Total        int64                       `json:"total"`
	Page         int                         `json:"page"`
	PageSize     int                         `json:"pageSize"`
}

// 交易统计响应
type TransactionStatsResponse struct {
	TotalCount     int64  `json:"totalCount"`
	PendingCount   int64  `json:"pendingCount"`
	SignedCount    int64  `json:"signedCount"`
	SubmittedCount int64  `json:"submittedCount"`
	ConfirmedCount int64  `json:"confirmedCount"`
	FailedCount    int64  `json:"failedCount"`
	TotalValue     string `json:"totalValue"`
	TodayCount     int64  `json:"todayCount"`
	WeekCount      int64  `json:"weekCount"`
	MonthCount     int64  `json:"monthCount"`
}
