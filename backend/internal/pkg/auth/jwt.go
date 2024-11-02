package auth

import (
	"crypto-custody/internal/model"
	"time"

	"github.com/golang-jwt/jwt"
)

// 用于处理 JWT 的认证与授权
type JWTAuth struct {
	secretKey     []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

// 创建一个新的 JWTAuth 实例，传入一个字符串类型的 secretKey 作为密钥
func NewJWTAuth(secretKey string) *JWTAuth {
	return &JWTAuth{
		secretKey:     []byte(secretKey),
		accessExpiry:  time.Hour,      // 访问令牌1小时过期
		refreshExpiry: time.Hour * 24, // 刷新令牌24小时过期
	}
}

// 用于生成访问令牌和刷新令牌。它接受一个 model.User 类型的参数 user，返回两个字符串（访问令牌和刷新令牌）和一个错误
func (j *JWTAuth) GenerateTokenPair(user *model.User) (string, string, error) {
	// 生成访问令牌
	accessToken, err := j.generateToken(user, j.accessExpiry, "access")
	if err != nil {
		return "", "", err
	}
	// 生成刷新令牌
	refreshToken, err := j.generateToken(user, j.refreshExpiry, "refresh")
	if err != nil {
		return "", "", err
	}
	return accessToken, refreshToken, nil
}

// 生成单个令牌, 传入用户信息, 过期时间, 令牌类型, 返回令牌字符串和错误
func (j *JWTAuth) generateToken(user *model.User, expiry time.Duration, tokenType string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,                       // 用户的唯一标识符
		"role":    user.Role,                     // 用户的角色
		"type":    tokenType,                     // 令牌的类型
		"exp":     time.Now().Add(expiry).Unix(), // 令牌的过期时间
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims) // 使用 HS256 签名方法创建新令牌
	return token.SignedString(j.secretKey)                     // 使用 secretKey 签名令牌并返回
}

// 验证传入的访问令牌字符串是否有效, 返回声明和错误
func (j *JWTAuth) ValidateAccessToken(tokenString string) (*jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return j.secretKey, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, jwt.ErrInvalidKey
	}
	if claims["type"] != "access" {
		return nil, jwt.ErrInvalidKey
	}
	return &claims, nil
}

// 验证传入的刷新令牌字符串是否有效, 返回声明和错误
func (j *JWTAuth) ValidateRefreshToken(tokenString string) (*jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return j.secretKey, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, jwt.ErrInvalidKey
	}
	if claims["type"] != "refresh" {
		return nil, jwt.ErrInvalidKey
	}
	return &claims, nil
}

// TODO: 实现 JWTAuth 的 RevokeRefreshToken 方法
// RevokeRefreshToken 方法接受一个刷新令牌字符串作为参数，用于撤销刷新令牌
// 该方法应该将传入的刷新令牌加入到一个撤销列表中，以防止被恶意使用
// 你可以使用一个全局变量来存储这个撤销列表，也可以使用其他方法
// 请注意，撤销列表中的刷新令牌应该在一定时间后过期，以避免无限制的增长
// 该方法应该返回一个错误，如果撤销成功则返回 nil
