package dto

type SeedTestDataRequest struct {
	Users        bool   `json:"users"`
	CaseCount    int    `json:"caseCount"`
	Accounts     int    `json:"accounts"`
	Transactions int    `json:"transactions"`
	CoinType     string `json:"coinType"`
}
