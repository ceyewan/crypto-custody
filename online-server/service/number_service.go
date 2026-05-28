package service

import (
	"fmt"
	"time"
)

func NewBusinessNo(prefix string) string {
	return fmt.Sprintf("%s-%s-%d", prefix, time.Now().Format("20060102150405"), time.Now().UnixNano()%1000000)
}
