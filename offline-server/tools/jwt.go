package tools

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	// jwtKey 是用于签名和验证JWT令牌的密钥
	// 注意: 在生产环境中，应该从环境变量或配置文件中加载此密钥
	jwtKey = []byte("your-jwt-secret-key")

	// blacklist 存储已撤销的令牌
	blacklist = make(map[string]time.Time)

	// blacklistMutex 保护黑名单的并发访问
	blacklistMutex sync.Mutex
)

// Claims 定义JWT令牌中包含的声明
type Claims struct {
	UserName string `json:"user_name"` // 用户名
	Role     string `json:"role"`      // 用户角色
	jwt.RegisteredClaims
}

// GenerateToken 生成带有过期时间的JWT令牌
//
// 参数:
//   - userName: 用户名
//   - role: 用户角色
//   - expiration: 令牌有效期
//
// 返回:
//   - string: 生成的JWT令牌字符串
//   - error: 如果生成过程中出现错误，返回相应错误；否则返回nil
func GenerateToken(userName string, role string, expiration time.Duration) (string, error) {
	expirationTime := jwt.NewNumericDate(time.Now().Add(expiration))

	claims := &Claims{
		UserName: userName,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: expirationTime,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		log.Printf("生成令牌签名失败: %v", err)
		return "", fmt.Errorf("生成令牌签名失败: %w", err)
	}

	log.Printf("已为用户: %s 生成令牌", userName)
	return tokenString, nil
}

// RevokeToken 将令牌加入黑名单
//
// 参数:
//   - tokenString: 要撤销的令牌字符串
//   - expiration: 黑名单中保存此令牌的时间（应与令牌过期时间一致）
func RevokeToken(tokenString string, expiration time.Duration) {
	blacklistMutex.Lock()
	defer blacklistMutex.Unlock()

	// 存储令牌及其过期时间
	blacklist[tokenString] = time.Now().Add(expiration)
	log.Printf("令牌已撤销: %s", tokenString)

	// 清理已过期的黑名单条目
	cleanBlacklist()
}

// cleanBlacklist 清理黑名单中已过期的令牌
// 此函数应该在持有blacklistMutex锁的情况下调用
func cleanBlacklist() {
	now := time.Now()
	for token, expiry := range blacklist {
		if now.After(expiry) {
			delete(blacklist, token)
		}
	}
}

// ValidateToken 验证令牌（包含过期检查和黑名单检查）
//
// 参数:
//   - tokenString: 要验证的令牌字符串
//
// 返回:
//   - string: 用户名，如果验证失败则为空字符串
//   - string: 用户角色，如果验证失败则为空字符串
//   - error: 如果验证过程中出现错误，返回相应错误；否则返回nil
func ValidateToken(tokenString string) (string, string, error) {
	// 检查令牌是否在黑名单中
	blacklistMutex.Lock()
	if expiry, found := blacklist[tokenString]; found {
		// 如果令牌已过期，从黑名单中移除
		if time.Now().After(expiry) {
			delete(blacklist, tokenString)
		} else {
			blacklistMutex.Unlock()
			log.Printf("令牌已被撤销: %s", tokenString)
			return "", "", fmt.Errorf("令牌已被撤销")
		}
	}
	blacklistMutex.Unlock()

	// 解析和验证令牌
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			log.Printf("意外的签名方法: %v", token.Header["alg"])
			return nil, fmt.Errorf("意外的签名方法: %v", token.Header["alg"])
		}
		return jwtKey, nil
	})

	if err != nil {
		log.Printf("解析令牌失败: %v", err)
		return "", "", fmt.Errorf("解析令牌失败: %w", err)
	}

	if !token.Valid {
		log.Printf("无效令牌: %s", tokenString)
		return "", "", fmt.Errorf("无效令牌")
	}

	log.Printf("已验证用户: %s 的令牌", claims.UserName)
	return claims.UserName, claims.Role, nil
}

// SetJWTKey 设置用于JWT操作的密钥
// 在生产环境中，应在应用启动时调用此函数设置密钥
//
// 参数:
//   - key: 用于JWT签名和验证的密钥
func SetJWTKey(key []byte) {
	if len(key) > 0 {
		jwtKey = key
		log.Printf("JWT密钥已更新")
	} else {
		log.Printf("警告: 尝试设置空的JWT密钥，操作被忽略")
	}
}
