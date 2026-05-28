package dto

type AuditLogListRequest struct {
	Page         int    `form:"page"`
	PageSize     int    `form:"pageSize"`
	Username     string `form:"username"`
	Action       string `form:"action"`
	ResourceType string `form:"resourceType"`
	CaseNo       string `form:"caseNo"`
	Result       string `form:"result"`
	StartTime    string `form:"startTime"`
	EndTime      string `form:"endTime"`
}
