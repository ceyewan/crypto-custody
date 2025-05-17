package service

import (
	"errors"
	"fmt"
	"online-server/model"
	"online-server/utils"
	"sync"
	"time"
)

var (
	userServiceInstance     *UserService
	userServiceInstanceOnce sync.Once
)

// UserService 提供用户相关的服务
type UserService struct {
	mu sync.RWMutex
}

// GetUserServiceInstance 获取用户服务实例
func GetUserServiceInstance() (*UserService, error) {
	userServiceInstanceOnce.Do(func() {
		userServiceInstance = &UserService{}
	})
	return userServiceInstance, nil
}

// Login 用户登录
func (s *UserService) Login(username, password string) (*model.User, string, error) {
	// 查询用户
	var user model.User
	if err := utils.GetDB().Where("username = ?", username).First(&user).Error; err != nil {
		return nil, "", errors.New("用户名或密码错误")
	}

	// 验证密码
	if !utils.CheckPasswordHash(password, user.Password) {
		return nil, "", errors.New("用户名或密码错误")
	}

	// 生成令牌
	token, err := utils.GenerateToken(user.Username, string(user.Role), time.Hour*24)
	if err != nil {
		return nil, "", fmt.Errorf("生成令牌失败: %w", err)
	}

	return &user, token, nil
}

// Register 用户注册
func (s *UserService) Register(username, password, email string) (*model.User, error) {
	// 检查用户名是否已存在
	var count int64
	utils.GetDB().Model(&model.User{}).Where("username = ?", username).Count(&count)
	if count > 0 {
		return nil, errors.New("用户名已存在")
	}

	// 检查邮箱是否已存在
	utils.GetDB().Model(&model.User{}).Where("email = ?", email).Count(&count)
	if count > 0 {
		return nil, errors.New("邮箱已被使用")
	}

	// 加密密码
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	// 创建用户
	user := model.User{
		Username: username,
		Password: hashedPassword,
		Email:    email,
		Role:     model.RoleGuest, // 默认为游客角色
	}

	if err := utils.GetDB().Create(&user).Error; err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	return &user, nil
}

// Logout 用户登出
func (s *UserService) Logout(token string) error {
	utils.RevokeToken(token, time.Hour*24) // 撤销令牌，保持在黑名单中 24 小时
	return nil
}

// GetUsers 获取所有用户
func (s *UserService) GetUsers() ([]model.User, error) {
	var users []model.User
	if err := utils.GetDB().Find(&users).Error; err != nil {
		return nil, fmt.Errorf("查询用户列表失败: %w", err)
	}
	return users, nil
}

// GetUserByID 根据ID获取用户
func (s *UserService) GetUserByID(id uint) (*model.User, error) {
	var user model.User
	if err := utils.GetDB().First(&user, id).Error; err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *UserService) GetUserByUsername(username string) (*model.User, error) {
	var user model.User
	if err := utils.GetDB().Where("username = ?", username).First(&user).Error; err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &user, nil
}

// UpdateUserRole 更新用户角色
func (s *UserService) UpdateUserRole(userID uint, role model.Role) error {
	// 首先检查目标用户是否为管理员
	var targetUser model.User
	if err := utils.GetDB().First(&targetUser, userID).Error; err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}

	// 不允许修改管理员用户的角色
	if targetUser.Role == model.RoleAdmin {
		return errors.New("不允许修改管理员用户的角色")
	}

	// 更新角色
	if err := utils.GetDB().Model(&model.User{}).Where("id = ?", userID).Update("role", role).Error; err != nil {
		return fmt.Errorf("更新用户角色失败: %w", err)
	}
	return nil
}

// ChangePassword 修改用户密码
func (s *UserService) ChangePassword(userID uint, oldPassword, newPassword string) error {
	// 获取用户
	var user model.User
	if err := utils.GetDB().First(&user, userID).Error; err != nil {
		return errors.New("用户不存在")
	}

	// 验证旧密码
	if !utils.CheckPasswordHash(oldPassword, user.Password) {
		return errors.New("原密码不正确")
	}

	// 加密新密码
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	// 更新密码
	if err := utils.GetDB().Model(&user).Update("password", hashedPassword).Error; err != nil {
		return fmt.Errorf("更新密码失败: %w", err)
	}

	return nil
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(userID uint) error {
	if err := utils.GetDB().Delete(&model.User{}, userID).Error; err != nil {
		return fmt.Errorf("删除用户失败: %w", err)
	}
	return nil
}

// UpdateUserID 更新用户名（用户ID）
// 注意：此方法仅允许管理员修改非管理员用户的用户名
func (s *UserService) UpdateUserID(userID uint, newUsername string) error {
	// 首先检查目标用户是否为管理员
	var targetUser model.User
	if err := utils.GetDB().First(&targetUser, userID).Error; err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}

	// 不允许修改管理员用户
	if targetUser.Role == model.RoleAdmin {
		return errors.New("不允许修改管理员用户的用户名")
	}

	// 检查新用户名是否已被使用
	var count int64
	utils.GetDB().Model(&model.User{}).Where("username = ? AND id != ?", newUsername, userID).Count(&count)
	if count > 0 {
		return errors.New("用户名已被使用")
	}

	// 更新用户名
	if err := utils.GetDB().Model(&model.User{}).Where("id = ?", userID).Update("username", newUsername).Error; err != nil {
		return fmt.Errorf("更新用户名失败: %w", err)
	}

	return nil
}
