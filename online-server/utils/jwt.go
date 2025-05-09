package utils

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte(os.Getenv("JWT_SECRET_KEY")) // 从环境变量获取密钥

// 定义 JWT 的 Claims 结构体
type Claims struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	RoleID   int    `json:"role_id"`
	jwt.RegisteredClaims
}

func GenerateJWT(userID int, username string, roleID int) (string, error) {
	// 设置 Token 的过期时间
	expirationTime := time.Now().Add(24 * time.Hour)

	// 创建 Claims
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RoleID:   roleID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime), // 过期时间
			IssuedAt:  jwt.NewNumericDate(time.Now()),     // 签发时间
			Issuer:    "your-app-name",                    // 签发者
		},
	}

	// 使用 HS256 签名算法创建 Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 使用密钥签名 Token
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// 验证 JWT
func ValidateJWT(tokenString string) (*Claims, error) {
	// 解析 Token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		return nil, err
	}

	// 验证 Token 是否有效
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
