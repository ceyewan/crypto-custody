package handler

import (
	"online-server/model"
	"online-server/utils"
	"strings"

	"github.com/gin-gonic/gin"
)

func ListJobs(c *gin.Context) {
	var jobs []model.Job
	utils.GetDB().Order("created_at DESC").Find(&jobs)
	utils.ResponseWithData(c, "查询批量任务成功", jobs)
}

func GetJob(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}
	var job model.Job
	if err := utils.GetDB().First(&job, id).Error; err != nil {
		utils.ResponseWithError(c, 404, "任务不存在")
		return
	}
	utils.ResponseWithData(c, "查询批量任务成功", job)
}

func JobResult(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	utils.ResponseWithData(c, "任务结果查询成功", gin.H{"jobId": id, "message": "当前任务无独立结果文件"})
}
