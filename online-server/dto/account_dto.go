package dto

// AccountRequest 是创建或更新账户时的请求体
type AccountRequest struct {
	Address         string `json:"address" binding:"required"`
	CoinType        string `json:"coinType" binding:"required"`
	AccountType     string `json:"accountType"`
	Balance         string `json:"balance"`
	BalanceSource   string `json:"balanceSource"`
	CaseNo          string `json:"caseNo"`
	Source          string `json:"source"`
	KeyMaterialHint string `json:"keyMaterialHint"`
	OfflineRefNo    string `json:"offlineRefNo"`
	Description     string `json:"description"`
}

// AccountResponse 是返回账户信息的响应体
type AccountResponse struct {
	Address         string `json:"address"`
	CoinType        string `json:"coinType"`
	AccountType     string `json:"accountType"`
	Balance         string `json:"balance"`
	BalanceSource   string `json:"balanceSource"`
	CaseNo          string `json:"caseNo"`
	ImportedBy      string `json:"importedBy"`
	Source          string `json:"source"`
	KeyMaterialHint string `json:"keyMaterialHint"`
	OfflineRefNo    string `json:"offlineRefNo"`
	Description     string `json:"description"`
}

// BatchImportRequest 是批量导入账户的请求体
type BatchImportRequest struct {
	Accounts []AccountRequest `json:"accounts" binding:"required"`
}

type AccountListRequest struct {
	Page        int    `form:"page"`
	PageSize    int    `form:"pageSize"`
	Address     string `form:"address"`
	CaseNo      string `form:"caseNo"`
	CoinType    string `form:"coinType"`
	AccountType string `form:"accountType"`
}
