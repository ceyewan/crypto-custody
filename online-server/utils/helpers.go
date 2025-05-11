package utils

import (
	"strconv"

	"gorm.io/gorm"
)

// Paginate 分页查询
func Paginate(page, pageSize string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		// 转换页码和每页数量
		pageNum, _ := strconv.Atoi(page)
		if pageNum <= 0 {
			pageNum = 1
		}

		pageSizeNum, _ := strconv.Atoi(pageSize)
		switch {
		case pageSizeNum > 100:
			pageSizeNum = 100
		case pageSizeNum <= 0:
			pageSizeNum = 10
		}

		// 计算偏移量
		offset := (pageNum - 1) * pageSizeNum
		return db.Offset(offset).Limit(pageSizeNum)
	}
}
