package service

import (
	"errors"
	"offline-server/storage/db"
	"offline-server/storage/model"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService 处理身份验证相关的业务逻辑
type AuthService struct{}

// NewAuthService 创建身份验证服务实例
func NewAuthService() *AuthService {
	return &AuthService{}
}

// Register 用户注册
func (s *AuthService) Register(username, password, email string) (*model.User, error) {
	database := db.GetDB()
	// 检查用户名是否已存在
	var existingUser model.User
	if err := database.Where("username = ?", username).First(&existingUser).Error; err == nil {
		return nil, errors.New("用户名已存在")
	} else if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 创建用户
	user := model.User{
		Username: username,
		Password: string(hashedPassword),
		Email:    email,
		Role:     "user", // 默认角色为普通用户
	}

	if err := database.Create(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// Login 用户登录
func (s *AuthService) Login(username, password string) (*model.User, error) {
	database := db.GetDB()
	var user model.User
	if err := database.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户名或密码错误")
		}
		return nil, err
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	return &user, nil
}

// GetUserByID 根据ID获取用户信息
func (s *AuthService) GetUserByID(id uint) (*model.User, error) {
	database := db.GetDB()
	var user model.User
	if err := database.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("用户不存在")
		}
		return nil, err
	}
	return &user, nil
}

// ListUsers 获取所有用户列表
func (s *AuthService) ListUsers() ([]model.User, error) {
	database := db.GetDB()
	var users []model.User
	if err := database.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// UpdateUserRole 更新用户角色
func (s *AuthService) UpdateUserRole(userID uint, role string) error {
	database := db.GetDB()
	return database.Model(&model.User{}).Where("id = ?", userID).Update("role", role).Error
}
