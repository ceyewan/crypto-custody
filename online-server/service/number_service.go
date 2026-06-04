package service

import (
	"fmt"
	"online-server/model"
	"online-server/utils"
	"time"
)

func NewTransactionNo() (string, error) {
	return nextDailyNo("TX", &model.Transaction{}, "tx_no")
}

func NewOfflineTaskNo() (string, error) {
	return nextDailyNo("TASK", &model.OfflineTask{}, "task_no")
}

func NewBusinessNo(prefix string) string {
	return fmt.Sprintf("%s-%s-%06d", prefix, time.Now().Format("20060102150405"), time.Now().UnixNano()%1000000)
}

func nextDailyNo(prefix string, modelRef interface{}, column string) (string, error) {
	datePart := time.Now().Format("20060102")
	base := fmt.Sprintf("%s-%s", prefix, datePart)

	for i := 1; i <= 999; i++ {
		number := fmt.Sprintf("%s-%03d", base, i)
		var count int64
		if err := utils.GetDB().Model(modelRef).Where(fmt.Sprintf("%s = ?", column), number).Count(&count).Error; err != nil {
			return "", fmt.Errorf("生成编号失败: %w", err)
		}
		if count == 0 {
			return number, nil
		}
	}

	return "", fmt.Errorf("当天编号已用尽: %s", base)
}
