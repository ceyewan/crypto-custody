// Package storage 提供对系统持久化数据的存储和访问管理
package storage

import (
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
func GetUserStorage() IUserStorage {
	userOnce.Do(func() {
		userInstance = &UserStorage{}
	})
	return userInstance
}

// CreateUser 创建新用户
func (s *UserStorage) CreateUser(username, password, email string) (*model.User, error) {
	if username == "" || password == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	// 检查用户名是否已存在
	var count int64
	if err := database.Model(&model.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		log.Printf("查询用户失败: %v", err)
		return nil, err
	}
	if count > 0 {
		return nil, ErrUserExists
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("密码加密失败: %v", err)
		return nil, err
	}

	// 创建用户
	user := model.User{
		Username: username,
		Password: string(hashedPassword),
		Email:    email,
		Role:     string(model.Guest), // 默认角色为普通用户
	}

	if err := database.Create(&user).Error; err != nil {
		log.Printf("创建用户失败: %v", err)
		return nil, err
	}

	return &user, nil
}

// GetUserByCredentials 通过用户名和密码获取用户
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
		return nil, err
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return &user, nil
}

// GetUserByID 根据ID获取用户信息
func (s *UserStorage) GetUserByID(id uint) (*model.User, error) {
	if id == 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var user model.User
	if err := database.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrUserNotFound
		}
		log.Printf("查询用户失败: %v", err)
		return nil, err
	}

	return &user, nil
}

// GetUserByUsername 根据用户名获取用户信息
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
		return nil, err
	}

	return &user, nil
}

// GetAllUsers 获取所有用户列表
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
		return nil, err
	}

	return users, nil
}

// UpdateUserRole 更新用户角色
func (s *UserStorage) UpdateUserRole(userID uint, role string) error {
	if userID == 0 {
		return ErrInvalidParameter
	}

	// 验证角色是否有效
	validRole := false
	switch model.Role(role) {
	case model.Admin, model.Coordinator, model.Participant, model.Guest:
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

	result := database.Model(&model.User{}).Where("id = ?", userID).Update("role", role)
	if result.Error != nil {
		log.Printf("更新用户角色失败: %v", result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrUserNotFound
	}

	return nil
}
