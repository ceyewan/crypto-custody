package dto

// AccountRequest 是创建或更新账户时的请求体
type AccountRequest struct {
	Address     string `json:"address" binding:"required"`
	CoinType    string `json:"coinType" binding:"required"`
	Description string `json:"description"`
}

// AccountResponse 是返回账户信息的响应体
type AccountResponse struct {
	Address     string `json:"address"`
	CoinType    string `json:"coinType"`
	Balance     string `json:"balance"`
	ImportedBy  string `json:"importedBy"`
	Description string `json:"description"`
}

// BatchImportRequest 是批量导入账户的请求体
type BatchImportRequest struct {
	Accounts []AccountRequest `json:"accounts" binding:"required"`
}
