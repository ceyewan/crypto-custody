package service

import (
	"encoding/json"
	"fmt"
	"online-server/model"
	"online-server/utils"

	"github.com/gin-gonic/gin"
)

// AuditAction records a security-relevant business operation.
func AuditAction(c *gin.Context, action, resourceType, resourceID, caseNo, result, errMsg string, afterData interface{}) {
	username := c.GetString("Username")
	role := c.GetString("Role")
	after := ""
	if afterData != nil {
		if b, err := json.Marshal(afterData); err == nil {
			after = string(b)
		}
	}
	log := model.AuditLog{
		RequestID:    c.GetHeader("X-Request-ID"),
		Username:     username,
		Role:         role,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		CaseNo:       caseNo,
		IP:           c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
		AfterData:    after,
		Result:       result,
		ErrorMessage: errMsg,
	}
	_ = utils.GetDB().Create(&log).Error
}

func ListAuditLogs(page, pageSize int, username, action, resourceType, caseNo, result string) ([]model.AuditLog, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	query := utils.GetDB().Model(&model.AuditLog{})
	if username != "" {
		query = query.Where("username = ?", username)
	}
	if action != "" {
		query = query.Where("action = ?", action)
	}
	if resourceType != "" {
		query = query.Where("resource_type = ?", resourceType)
	}
	if caseNo != "" {
		query = query.Where("case_no = ?", caseNo)
	}
	if result != "" {
		query = query.Where("result = ?", result)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询审计日志总数失败: %w", err)
	}
	var logs []model.AuditLog
	if err := query.Order("created_at DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("查询审计日志失败: %w", err)
	}
	return logs, total, nil
}
