package dto

type CaseRequest struct {
	CaseNo      string `json:"caseNo"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

type BatchImportCasesRequest struct {
	Cases []CaseRequest `json:"cases" binding:"required"`
}

type CaseListRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	CaseNo   string `form:"caseNo"`
	Keyword  string `form:"keyword"`
	Status   string `form:"status"`
}

type LinkAccountRequest struct {
	AccountID uint `json:"accountId" binding:"required"`
}

type CustodyWalletResultRequest struct {
	TaskNo         string `json:"taskNo"`
	CaseNo         string `json:"caseNo"`
	CoinType       string `json:"coinType"`
	CustodyAddress string `json:"custodyAddress" binding:"required"`
	PublicKey      string `json:"publicKey"`
	OfflineRefNo   string `json:"offlineRefNo"`
	CompletedAt    string `json:"completedAt"`
}
