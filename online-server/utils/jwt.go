package utils

import (
	"fmt"
	"sync"
	"time"

	"github.com/ceyewan/clog"
	"github.com/golang-jwt/jwt/v5"
)

var (
	// jwtKey 是用于签名和验证JWT令牌的密钥
	// 注意: 在生产环境中，应该从环境变量或配置文件中加载此密钥
	jwtKey = []byte("your-jwt-secret-key")

	// blacklist 存储已撤销的令牌
	blacklist = make(map[string]time.Time)

	// blacklistMutex 保护黑名单的并发访问，使用读写锁提高并发性能
	blacklistMutex sync.RWMutex
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
	logger := clog.Module("jwt")
	logger.Info("开始生成JWT令牌", clog.String("username", userName), clog.String("role", role))
	
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
		logger.Error("生成令牌签名失败", clog.Err(err), clog.String("username", userName))
		return "", fmt.Errorf("生成令牌签名失败: %w", err)
	}

	logger.Info("JWT令牌生成成功", 
		clog.String("username", userName), 
		clog.String("expires", expirationTime.Time.Format(time.RFC3339)))
	return tokenString, nil
}

// RevokeToken 将令牌加入黑名单
//
// 参数:
//   - tokenString: 要撤销的令牌字符串
//   - expiration: 黑名单中保存此令牌的时间（应与令牌过期时间一致）
func RevokeToken(tokenString string, expiration time.Duration) {
	logger := clog.Module("jwt")
	
	// 尝试解析令牌以获取用户信息（不验证有效性）
	token, _ := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	
	// 获取用户信息（如果能解析）
	var username string
	if token != nil {
		if claims, ok := token.Claims.(*Claims); ok {
			username = claims.UserName
			logger.Info("准备撤销令牌", clog.String("username", username))
		}
	}

	// 存储令牌及其过期时间
	expiryTime := time.Now().Add(expiration)
	
	// 使用写锁添加令牌到黑名单
	blacklistMutex.Lock()
	blacklist[tokenString] = expiryTime
	blacklistMutex.Unlock()
	
	if username != "" {
		logger.Info("令牌已撤销", 
			clog.String("username", username),
			clog.String("expiry", expiryTime.Format(time.RFC3339)))
	} else {
		logger.Info("令牌已撤销", clog.String("token_prefix", tokenString[:10]+"..."))
	}

	// 异步清理已过期的黑名单条目，避免阻塞当前请求
	go cleanExpiredTokens()
}

// cleanExpiredTokens 清理黑名单中已过期的令牌
// 独立的函数，可以异步调用以减少锁持有时间
func cleanExpiredTokens() {
	logger := clog.Module("jwt")
	now := time.Now()
	
	// 创建过期令牌列表，先读取后批量删除，减少锁竞争
	var expiredTokens []string
	
	// 使用读锁识别过期令牌
	blacklistMutex.RLock()
	for token, expiry := range blacklist {
		if now.After(expiry) {
			expiredTokens = append(expiredTokens, token)
		}
	}
	blacklistMutex.RUnlock()
	
	// 如果有过期令牌，使用写锁删除
	count := len(expiredTokens)
	if count > 0 {
		blacklistMutex.Lock()
		for _, token := range expiredTokens {
			// 再次检查过期时间，以防在获取写锁期间被其他协程修改
			if expiry, exists := blacklist[token]; exists && now.After(expiry) {
				delete(blacklist, token)
			} else {
				count-- // 如果令牌不存在或已更新，则调整计数
			}
		}
		remaining := len(blacklist)
		blacklistMutex.Unlock()
		
		logger.Info("清理过期的黑名单令牌", 
			clog.Int("cleaned", count), 
			clog.Int("remaining", remaining))
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
	logger := clog.Module("jwt")
	
	// 检查令牌是否在黑名单中（使用读锁提高并发性能）
	now := time.Now()
	blacklistMutex.RLock()
	expiry, found := blacklist[tokenString]
	blacklistMutex.RUnlock()
	
	if found {
		// 如果令牌已过期，从黑名单中移除
		if now.After(expiry) {
			// 使用写锁删除过期令牌
			blacklistMutex.Lock()
			// 双重检查，确保令牌仍在黑名单中且仍然过期
			if currentExpiry, stillExists := blacklist[tokenString]; stillExists && now.After(currentExpiry) {
				delete(blacklist, tokenString)
				logger.Info("从黑名单中移除过期令牌")
			}
			blacklistMutex.Unlock()
		} else {
			logger.Warn("尝试使用已撤销的令牌", clog.String("token_prefix", tokenString[:10]+"..."))
			return "", "", fmt.Errorf("令牌已被撤销")
		}
	}

	// 解析和验证令牌
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			logger.Error("意外的签名方法", clog.Any("alg", token.Header["alg"]))
			return nil, fmt.Errorf("意外的签名方法: %v", token.Header["alg"])
		}
		return jwtKey, nil
	})

	if err != nil {
		logger.Warn("解析令牌失败", clog.Err(err))
		return "", "", fmt.Errorf("解析令牌失败: %w", err)
	}

	if !token.Valid {
		logger.Warn("无效令牌", clog.String("token_prefix", tokenString[:10]+"..."))
		return "", "", fmt.Errorf("无效令牌")
	}

	// 检查令牌是否即将过期
	if claims.ExpiresAt != nil {
		remaining := time.Until(claims.ExpiresAt.Time)
		if remaining < 10*time.Minute {
			logger.Warn("令牌即将过期", 
				clog.String("username", claims.UserName), 
				clog.Duration("remaining", remaining))
		} else if remaining < 30*time.Minute && remaining > 10*time.Minute {
			// 当令牌剩余时间在10-30分钟之间时，仅记录信息级别日志
			logger.Info("令牌将在不久后过期",
				clog.String("username", claims.UserName),
				clog.Duration("remaining", remaining))
		}
	}

	logger.Info("令牌验证成功", 
		clog.String("username", claims.UserName), 
		clog.String("role", claims.Role))
	return claims.UserName, claims.Role, nil
}

// SetJWTKey 设置用于JWT操作的密钥
// 在生产环境中，应在应用启动时调用此函数设置密钥
//
// 参数:
//   - key: 用于JWT签名和验证的密钥
func SetJWTKey(key []byte) {
	logger := clog.Module("jwt")
	
	if len(key) > 0 {
		jwtKey = key
		logger.Info("JWT密钥已更新", clog.Int("key_length", len(key)))
	} else {
		logger.Warn("尝试设置空的JWT密钥，操作被忽略")
	}
}

// CheckAuth 检查Token是否有效
//
// 参数:
//   - tokenString: 要验证的令牌字符串
//
// 返回:
//   - bool: 令牌是否有效
//   - string: 用户名，如果验证失败则为空字符串
//   - string: 用户角色，如果验证失败则为空字符串
func CheckAuth(tokenString string) (bool, string, string) {
	logger := clog.Module("jwt")
	
	// 验证令牌
	username, role, err := ValidateToken(tokenString)
	if err != nil {
		logger.Warn("令牌验证失败", 
			clog.Err(err), 
			clog.String("token_prefix", tokenString[:10]+"..."))
		return false, "", ""
	}
	
	return true, username, role
}
