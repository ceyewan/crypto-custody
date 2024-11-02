// service/user/service.go
package service

import (
	"context"
	"crypto-custody/internal/api/dto"
	"crypto-custody/internal/model"
	"crypto-custody/internal/pkg/auth"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserService struct {
	db      *gorm.DB
	jwtAuth *auth.JWTAuth
}

func NewUserService(db *gorm.DB, jwtAuth *auth.JWTAuth) *UserService { // 修改构造函数名称
	return &UserService{db: db, jwtAuth: jwtAuth}
}

func (s *UserService) Register(ctx context.Context, req dto.RegisterRequest) (*model.User, error) {
	// 检查用户是否已存在
	var count int64
	if err := s.db.Model(&model.User{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, errors.New("username already exists")
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 创建用户
	user := &model.User{
		Username: req.Username,
		Password: string(hashedPassword),
		// Email:    req.Email,
		// Phone:    req.Phone,
		Role: "user",
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) Login(ctx context.Context, req dto.LoginRequest) (*dto.TokenResponse, error) {
	// 查找用户
	var user model.User
	if err := s.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
		return nil, errors.New("invalid credentials")
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// 生成token
	accessToken, refreshToken, err := s.jwtAuth.GenerateTokenPair(&user)
	if err != nil {
		return nil, err
	}

	return &dto.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600, // token有效期（秒）
	}, nil
}

func (s *UserService) RefreshToken(ctx context.Context, refreshToken string) (*dto.TokenResponse, error) {
	// 验证refresh token
	// claims, err := s.jwtAuth.ValidateRefreshToken(refreshToken)
	// if err != nil {
	// 	return nil, err
	// }

	// 获取用户信息
	// var user model.User
	// if err := s.db.First(&user, claims.UserID).Error; err != nil {
	// 	return nil, err
	// }

	// 生成新的token对
	// accessToken, newRefreshToken, err := s.jwtAuth.GenerateTokenPair(&user)
	// if err != nil {
	// 	return nil, err
	// }

	// return &dto.TokenResponse{
	// 	AccessToken:  accessToken,
	// 	RefreshToken: newRefreshToken,
	// 	TokenType:    "Bearer",
	// 	ExpiresIn:    3600,
	// }, nil
	// todo
}

func (s *UserService) CreateUser(username, password, role string) error {
	hashedPassword := hashPassword(password)
	user := &model.User{
		Username: username,
		Password: hashedPassword,
		Role:     role,
	}
	return s.db.Create(user).Error
}

func (s *UserService) ValidateUser(username, password string) (*model.User, error) {
	var user model.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}

	if user.Password != hashPassword(password) {
		return nil, errors.New("invalid password")
	}

	return &user, nil
}

func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}
