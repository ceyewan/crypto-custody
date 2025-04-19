package service

import (
	"offline-server/storage"
	"offline-server/storage/model"
)

// AuthService 处理身份验证相关的业务逻辑
type AuthService struct {
	userStorage storage.IUserStorage
}

// NewAuthService 创建身份验证服务实例
func NewAuthService() *AuthService {
	return &AuthService{
		userStorage: storage.GetUserStorage(),
	}
}

// Register 用户注册
func (s *AuthService) Register(username, password, email string) (*model.User, error) {
	return s.userStorage.CreateUser(username, password, email)
}

// Login 用户登录
func (s *AuthService) Login(username, password string) (*model.User, error) {
	return s.userStorage.GetUserByCredentials(username, password)
}

// GetUserByID 根据ID获取用户信息
func (s *AuthService) GetUserByID(id uint) (*model.User, error) {
	return s.userStorage.GetUserByID(id)
}

// ListUsers 获取所有用户列表
func (s *AuthService) ListUsers() ([]model.User, error) {
	return s.userStorage.GetAllUsers()
}

// UpdateUserRole 更新用户角色
func (s *AuthService) UpdateUserRole(userID uint, role string) error {
	return s.userStorage.UpdateUserRole(userID, role)
}
