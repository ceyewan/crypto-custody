package model

import "gorm.io/gorm"

// Job 记录批量导入、余额同步等批量任务。
type Job struct {
	gorm.Model
	JobNo        string `gorm:"column:job_no;uniqueIndex;size:80;not null;comment:任务编号"`
	Type         string `gorm:"column:type;size:60;index;comment:任务类型"`
	Status       string `gorm:"column:status;size:30;index;comment:任务状态"`
	Total        int    `gorm:"column:total;comment:总数"`
	Success      int    `gorm:"column:success;comment:成功数"`
	Failed       int    `gorm:"column:failed;comment:失败数"`
	Progress     int    `gorm:"column:progress;comment:进度百分比"`
	ResultFile   string `gorm:"column:result_file;size:500;comment:结果文件"`
	CreatedBy    string `gorm:"column:created_by;size:80;comment:创建人"`
	StartedAt    *int64 `gorm:"column:started_at;comment:开始时间Unix秒"`
	FinishedAt   *int64 `gorm:"column:finished_at;comment:完成时间Unix秒"`
	ErrorMessage string `gorm:"column:error_message;type:text;comment:错误摘要"`
}
