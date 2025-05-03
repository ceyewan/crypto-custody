package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"offline-server/storage"
)

// keyGenStorage 密钥生成会话存储接口
var keyGenStorage storage.IKeyGenStorage = storage.GetKeyGenStorage()

// CreateKenGenSessionKey 创建密钥生成会话密钥
func CreateKenGenSessionKey(initiator string) (string, error) {
	// 参数验证
	if strings.TrimSpace(initiator) == "" {
		return "", errors.New("发起者不能为空")
	}

	// 生成会话密钥
	timestamp := time.Now().Format("20060102150405")
	sessionKey := fmt.Sprintf("keygen_%s_%s", timestamp, initiator)

	return sessionKey, nil
}
