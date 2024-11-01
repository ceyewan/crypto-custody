package auth

import (
	"crypto-custody/internal/model"
	"time"

	"github.com/golang-jwt/jwt"
)

type JWTAuth struct {
	secretKey     []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewJWTAuth(secretKey string) *JWTAuth {
	return &JWTAuth{
		secretKey:     []byte(secretKey),
		accessExpiry:  time.Hour,      // 访问令牌1小时过期
		refreshExpiry: time.Hour * 24, // 刷新令牌24小时过期
	}
}

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

func (j *JWTAuth) generateToken(user *model.User, expiry time.Duration, tokenType string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role,
		"type":    tokenType,
		"exp":     time.Now().Add(expiry).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secretKey)
}

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
