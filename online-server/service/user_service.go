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

// UserService 提供用户相关的服务，包括用户认证、注册、信息管理等功能
type UserService struct {
	mu sync.RWMutex // 读写锁，用于并发操作保护
}

// GetUserServiceInstance 获取UserService的单例实例
//
// 确保整个应用程序中只有一个UserService实例存在
//
// 返回：
// - *UserService：UserService的单例实例
// - error：实例化过程中发生的错误（如有）
func GetUserServiceInstance() (*UserService, error) {
	userServiceInstanceOnce.Do(func() {
		userServiceInstance = &UserService{}
	})
	return userServiceInstance, nil
}

// Login 用户登录认证
//
// 验证用户提供的用户名和密码，成功时返回用户对象和JWT令牌
//
// 参数：
// - username：尝试登录的用户名
// - password：明文密码（将与数据库中的哈希值进行比较）
//
// 返回：
// - *model.User：认证成功的用户信息
// - string：用于已认证会话的JWT令牌
// - error：认证过程中的错误（如有）
func (s *UserService) Login(username, password string) (*model.User, string, error) {
	// 根据用户名查询用户
	var user model.User
	if err := utils.GetDB().Where("username = ?", username).First(&user).Error; err != nil {
		return nil, "", errors.New("用户名或密码错误")
	}

	// 验证密码哈希值
	if !utils.CheckPasswordHash(password, user.Password) {
		return nil, "", errors.New("用户名或密码错误")
	}

	// 生成JWT令牌，有效期24小时
	token, err := utils.GenerateToken(user.Username, string(user.Role), time.Hour*24)
	if err != nil {
		return nil, "", fmt.Errorf("生成令牌失败: %w", err)
	}

	return &user, token, nil
}

// Register 用户注册
//
// 使用提供的信息创建新用户账户，新用户默认被分配游客角色
//
// 参数：
// - username：新账户的用户名（必须唯一）
// - password：新账户的密码（将被哈希处理）
// - email：新账户的电子邮件地址（必须唯一）
//
// 返回：
// - *model.User：新创建的用户记录
// - error：注册过程中的错误（例如，用户名重复）
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

	// 哈希处理密码
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	// 创建新用户，默认为游客角色
	user := model.User{
		Username: username,
		Password: hashedPassword,
		Email:    email,
		Role:     model.RoleGuest, // 默认角色为游客
	}

	if err := utils.GetDB().Create(&user).Error; err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	return &user, nil
}

// Logout 用户登出
//
// 通过将JWT令牌加入黑名单来使用户会话无效
// 被吊销的令牌在黑名单中保留24小时
//
// 参数：
// - token：要吊销的JWT令牌
//
// 返回：
// - error：登出过程中发生的错误（如有）
func (s *UserService) Logout(token string) error {
	utils.RevokeToken(token, time.Hour*24) // 令牌在黑名单中保留24小时
	return nil
}

// GetUsers 获取所有用户
//
// 从数据库中检索所有用户记录
//
// 返回：
// - []model.User：包含所有用户记录的切片
// - error：数据库查询过程中发生的错误（如有）
func (s *UserService) GetUsers() ([]model.User, error) {
	var users []model.User
	if err := utils.GetDB().Find(&users).Error; err != nil {
		return nil, fmt.Errorf("查询用户列表失败: %w", err)
	}
	return users, nil
}

// GetUserByID 根据ID获取用户
//
// 通过唯一标识符检索特定用户
//
// 参数：
// - id：要检索的用户的唯一标识符
//
// 返回：
// - *model.User：如果找到，则返回用户记录
// - error：数据库查询过程中发生的错误（如有）
func (s *UserService) GetUserByID(id uint) (*model.User, error) {
	var user model.User
	if err := utils.GetDB().First(&user, id).Error; err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
//
// 通过用户名检索特定用户
//
// 参数：
// - username：要检索的用户的用户名
//
// 返回：
// - *model.User：如果找到，则返回用户记录
// - error：数据库查询过程中发生的错误（如有）
func (s *UserService) GetUserByUsername(username string) (*model.User, error) {
	var user model.User
	if err := utils.GetDB().Where("username = ?", username).First(&user).Error; err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &user, nil
}

// UpdateUserRole 更新用户角色
//
// 将用户的角色更改为指定的角色类型
// 管理员用户的角色不能被修改
//
// 参数：
// - userID：要更新角色的用户ID
// - role：要分配给用户的新角色
//
// 返回：
// - error：角色更新过程中发生的错误（如有）
func (s *UserService) UpdateUserRole(userID uint, role model.Role) error {
	// 检查目标用户是否存在
	var targetUser model.User
	if err := utils.GetDB().First(&targetUser, userID).Error; err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}

	// 防止修改管理员用户的角色
	if targetUser.Role == model.RoleAdmin {
		return errors.New("不允许修改管理员用户的角色")
	}

	// 更新用户角色
	if err := utils.GetDB().Model(&model.User{}).Where("id = ?", userID).Update("role", role).Error; err != nil {
		return fmt.Errorf("更新用户角色失败: %w", err)
	}
	return nil
}

// ChangePassword 修改用户密码
//
// 在验证旧密码后更新用户的密码
//
// 参数：
// - userID：更改密码的用户ID
// - oldPassword：当前密码，用于验证
// - newPassword：要设置的新密码
//
// 返回：
// - error：密码更新过程中发生的错误（如有）
func (s *UserService) ChangePassword(userID uint, oldPassword, newPassword string) error {
	// 获取用户
	var user model.User
	if err := utils.GetDB().First(&user, userID).Error; err != nil {
		return errors.New("用户不存在")
	}

	// 验证当前密码
	if !utils.CheckPasswordHash(oldPassword, user.Password) {
		return errors.New("原密码不正确")
	}

	// 哈希处理新密码
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
//
// 通过ID从数据库中删除用户
//
// 参数：
// - userID：要删除的用户ID
//
// 返回：
// - error：用户删除过程中发生的错误（如有）
func (s *UserService) DeleteUser(userID uint) error {
	if err := utils.GetDB().Delete(&model.User{}, userID).Error; err != nil {
		return fmt.Errorf("删除用户失败: %w", err)
	}
	return nil
}

// UpdateUserID 更新用户名
//
// 将用户的用户名更改为新值
// 管理员用户的用户名不能被修改
//
// 参数：
// - userID：要更改用户名的用户ID
// - newUsername：要设置的新用户名
//
// 返回：
// - error：用户名更新过程中发生的错误（如有）
func (s *UserService) UpdateUserID(userID uint, newUsername string) error {
	// 检查目标用户是否存在
	var targetUser model.User
	if err := utils.GetDB().First(&targetUser, userID).Error; err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}

	// 防止修改管理员用户名
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
