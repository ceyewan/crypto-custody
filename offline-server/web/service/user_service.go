package service

import (
	"errors"
	"offline-server/storage"
	"offline-server/storage/db"
	"offline-server/storage/model"
	"strings"
)

// userStorage 用户存储接口
var userStorage storage.IUserStorage = storage.GetUserStorage()

// LoginUser 用户登录服务
func LoginUser(username, password string) (*model.User, error) {
	// 输入验证
	if strings.TrimSpace(username) == "" || strings.TrimSpace(password) == "" {
		return nil, errors.New("用户名和密码不能为空")
	}

	// 调用存储接口验证用户凭证
	user, err := userStorage.GetUserByCredentials(username, password)
	if err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	return user, nil
}

// RegisterUser 用户注册服务
func RegisterUser(username, password, email string) (*model.User, error) {
	// 输入验证
	if strings.TrimSpace(username) == "" {
		return nil, errors.New("用户名不能为空")
	}
	if strings.TrimSpace(password) == "" {
		return nil, errors.New("密码不能为空")
	}
	if strings.TrimSpace(email) == "" {
		return nil, errors.New("邮箱不能为空")
	}

	// 调用存储接口创建用户
	user, err := userStorage.CreateUser(username, password, email)
	if err != nil {
		// 根据错误类型返回友好的错误信息
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, errors.New("用户名已存在")
		}
		return nil, errors.New("用户注册失败: " + err.Error())
	}

	return user, nil
}

// GetUserByUserName 根据用户名获取用户信息
func GetUserByUserName(username string) (*model.User, error) {
	// 输入验证
	if strings.TrimSpace(username) == "" {
		return nil, errors.New("用户名不能为空")
	}

	// 调用存储接口获取用户信息
	user, err := userStorage.GetUserByUsername(username)
	if err != nil {
		return nil, errors.New("用户不存在")
	}

	return user, nil
}

// GetAllUsers 获取所有用户列表
func GetAllUsers() ([]model.User, error) {
	return userStorage.GetAllUsers()
}

// UpdateUserRole 更新用户角色
func UpdateUserRole(userName, role string) error {
	// 验证用户是否存在
	user, err := userStorage.GetUserByUsername(userName)
	if err != nil {
		return errors.New("用户不存在")
	}

	// 验证角色是否有效（使用 model.Role 类型）
	isValid := false
	validRoles := []model.Role{model.RoleAdmin, model.RoleCoordinator, model.RoleParticipant, model.RoleGuest}
	for _, validRole := range validRoles {
		if role == string(validRole) {
			isValid = true
			break
		}
	}

	if !isValid {
		return errors.New("无效的角色")
	}

	// 防止用户降级自己的admin权限
	if user.Role == model.RoleAdmin && role != string(model.RoleAdmin) {
		// 检查是否至少还有一个管理员
		users, err := userStorage.GetAllUsers()
		if err != nil {
			return err
		}

		adminCount := 0
		for _, u := range users {
			if u.Role == model.RoleAdmin && u.Username != userName {
				adminCount++
			}
		}

		if adminCount == 0 {
			return errors.New("系统需要至少一个管理员账户")
		}
	}

	// 调用存储接口更新用户角色
	return userStorage.UpdateUserRole(userName, role)
}

// GetAvailableUsers 获取可以参与密钥生成的用户列表（协调者和参与者）
func GetAvailableUsers() ([]model.User, error) {
	users, err := userStorage.GetAllUsers()
	if err != nil {
		return nil, err
	}

	// 筛选出只有协调者和参与者角色的用户
	availableUsers := []model.User{}
	for _, user := range users {
		if user.Role == model.RoleCoordinator || user.Role == model.RoleParticipant {
			availableUsers = append(availableUsers, user)
		}
	}

	return availableUsers, nil
}

// GetUsersByAddress 获取拥有特定地址分片的用户列表
func GetUsersByAddress(address string) ([]model.User, error) {
	if strings.TrimSpace(address) == "" {
		return nil, errors.New("以太坊地址不能为空")
	}

	// 获取DB实例
	database := db.GetDB()
	if database == nil {
		return nil, errors.New("数据库未初始化")
	}

	// 查询拥有该地址分片的所有用户名
	var usernames []string
	if err := database.Model(&model.EthereumKeyShard{}).
		Where("address = ?", address).
		Pluck("username", &usernames).Error; err != nil {
		return nil, errors.New("查询密钥分片失败: " + err.Error())
	}

	if len(usernames) == 0 {
		return []model.User{}, nil
	}

	// 获取这些用户的详细信息
	var users []model.User
	if err := database.Where("username IN ?", usernames).Find(&users).Error; err != nil {
		return nil, errors.New("查询用户信息失败: " + err.Error())
	}

	return users, nil
}
