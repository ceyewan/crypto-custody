package service

import (
	"errors"
	"fmt"
	"offline-server/storage"
	"strings"
	"time"
)

// signStorage 签名会话存储接口
var signStorage storage.ISignStorage = storage.GetSignStorage()

// CreateSignSessionKey 创建签名会话密钥
func CreateSignSessionKey(initiator string) (string, error) {
	// 参数验证
	if strings.TrimSpace(initiator) == "" {
		return "", errors.New("发起者不能为空")
	}

	// 生成会话密钥
	timestamp := time.Now().Format("20060102150405")
	sessionKey := fmt.Sprintf("sign_%s_%s", timestamp, initiator)

	return sessionKey, nil
}
