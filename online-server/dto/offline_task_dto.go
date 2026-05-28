package dto

type CustodyKeygenTaskRequest struct {
	CaseID          uint   `json:"caseId" binding:"required"`
	CoinType        string `json:"coinType"`
	ThresholdPolicy string `json:"thresholdPolicy"`
}

type ImportOfflineResultRequest struct {
	Result map[string]interface{} `json:"result" binding:"required"`
}

type OfflineTaskListRequest struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	TaskType string `form:"taskType"`
	CaseNo   string `form:"caseNo"`
	Status   string `form:"status"`
}
