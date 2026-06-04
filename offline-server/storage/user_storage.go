// Package storage 提供对系统持久化数据的存储和访问管理
package storage

import (
	"fmt"
	"log"
	"sync"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"offline-server/storage/db"
	"offline-server/storage/model"
)

// UserStorage 提供对用户账号的存储和访问
type UserStorage struct {
	mu sync.RWMutex // 使用读写锁提高并发效率
}

var (
	userInstance *UserStorage
	userOnce     sync.Once
)

// GetUserStorage 返回 UserStorage 的单例实例
// 通过单例模式确保整个应用程序中只有一个存储实例
func GetUserStorage() IUserStorage {
	userOnce.Do(func() {
		userInstance = &UserStorage{}
	})
	return userInstance
}

// CreateUser 创建新用户
// 参数：
//   - username: 登录标识，唯一标识一个用户
//   - password: 用户密码，会被加密存储
//   - nickname: 昵称或姓名
//
// 返回：
//   - 创建的用户对象指针
//   - 如果创建失败则返回错误信息
func (s *UserStorage) CreateUser(username, password, nickname string) (*model.User, error) {
	if username == "" || password == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	email := legacyEmailForIdentifier(username)

	// 检查登录标识是否已存在。Email 是历史字段，只保存自动生成值。
	var count int64
	if err := database.Model(&model.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		log.Printf("查询用户失败: %v", err)
		return nil, ErrOperationFailed
	}
	if count > 0 {
		return nil, ErrUserExists
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("密码加密失败: %v", err)
		return nil, ErrOperationFailed
	}

	// 创建用户
	user := model.User{
		Username: username,
		Nickname: nickname,
		Password: string(hashedPassword),
		Email:    email,
		Role:     model.RoleOfficer,
		Status:   model.UserStatusActive,
	}

	if err := database.Create(&user).Error; err != nil {
		log.Printf("创建用户失败: %v", err)
		return nil, ErrOperationFailed
	}

	return &user, nil
}

func legacyEmailForIdentifier(identifier string) string {
	return fmt.Sprintf("%s@offline.local", identifier)
}

// GetUserByCredentials 通过用户名和密码获取用户
// 参数：
//   - username: 登录标识
//   - password: 用户密码（明文）
//
// 返回：
//   - 匹配的用户对象指针
//   - 如果用户不存在或密码错误则返回错误信息
func (s *UserStorage) GetUserByCredentials(username, password string) (*model.User, error) {
	if username == "" || password == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var user model.User
	if err := database.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrInvalidCredentials
		}
		log.Printf("查询用户失败: %v", err)
		return nil, ErrOperationFailed
	}

	if user.Status != model.UserStatusActive {
		return nil, ErrUserDisabled
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return &user, nil
}

// GetUserByUsername 根据用户名获取用户信息
// 参数：
//   - username: 登录标识
//
// 返回：
//   - 匹配的用户对象指针
//   - 如果用户不存在则返回错误信息
func (s *UserStorage) GetUserByUsername(username string) (*model.User, error) {
	if username == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var user model.User
	if err := database.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrUserNotFound
		}
		log.Printf("查询用户失败: %v", err)
		return nil, ErrOperationFailed
	}

	return &user, nil
}

// GetAllUsers.获取所有用户列表
// 返回：
//   - 用户列表数组
//   - 如果查询失败则返回错误信息
func (s *UserStorage) GetAllUsers() ([]model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var users []model.User
	if err := database.Find(&users).Error; err != nil {
		log.Printf("查询所有用户失败: %v", err)
		return nil, ErrOperationFailed
	}

	return users, nil
}

// UpdateUserRole 更新用户角色
// 参数：
//   - username: 用户名
//   - role: 新的角色，必须是系统预定义的角色之一
//
// 返回：
//   - 如果更新失败则返回错误信息
func (s *UserStorage) UpdateUserRole(username string, role string) error {
	if username == "" || role == "" {
		return ErrInvalidParameter
	}

	// 验证角色是否有效
	validRole := false
	switch model.Role(role) {
	case model.RoleAdmin, model.RoleOfficer, model.RoleAuditor:
		validRole = true
	}

	if !validRole {
		return ErrInvalidRole
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	result := database.Model(&model.User{}).Where("username = ?", username).Update("role", role)
	if result.Error != nil {
		log.Printf("更新用户角色失败: %v", result.Error)
		return ErrOperationFailed
	}

	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}

// UpdateUserStatus 更新用户状态
func (s *UserStorage) UpdateUserStatus(username string, status model.UserStatus) error {
	if username == "" {
		return ErrInvalidParameter
	}
	if status != model.UserStatusActive && status != model.UserStatusDisabled {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	result := database.Model(&model.User{}).Where("username = ?", username).Update("status", status)
	if result.Error != nil {
		log.Printf("更新用户状态失败: %v", result.Error)
		return ErrOperationFailed
	}
	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}
