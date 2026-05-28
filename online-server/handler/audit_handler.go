package handler

import (
	"net/http"
	"online-server/dto"
	"online-server/model"
	"online-server/service"
	"online-server/utils"

	"github.com/gin-gonic/gin"
)

func ListAuditLogs(c *gin.Context) {
	var req dto.AuditLogListRequest
	_ = c.ShouldBindQuery(&req)
	logs, total, err := service.ListAuditLogs(req.Page, req.PageSize, req.Username, req.Action, req.ResourceType, req.CaseNo, req.Result)
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	utils.ResponseWithData(c, "查询审计日志成功", gin.H{"items": logs, "total": total, "page": req.Page, "pageSize": req.PageSize})
}

func GetAuditLog(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}
	var log model.AuditLog
	if err := utils.GetDB().First(&log, id).Error; err != nil {
		utils.ResponseWithError(c, http.StatusNotFound, "审计日志不存在")
		return
	}
	utils.ResponseWithData(c, "查询审计日志成功", log)
}

func ExportAuditLogs(c *gin.Context) {
	var logs []model.AuditLog
	if err := utils.GetDB().Order("created_at DESC").Limit(10000).Find(&logs).Error; err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "导出审计日志失败: "+err.Error())
		return
	}
	service.AuditAction(c, "audit.export", "audit_log", "", "", "success", "", gin.H{"count": len(logs)})
	utils.ResponseWithData(c, "审计日志导出成功", logs)
}
