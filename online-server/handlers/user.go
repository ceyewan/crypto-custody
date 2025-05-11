package handlers

import (
	"net/http"
	"online-server/model"
	"online-server/utils"
	"time"

	"github.com/ceyewan/clog"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Login 用户登录
func Login(c *gin.Context) {
	logger := clog.Module("user")
	logger.Info("处理登录请求", clog.String("ip", c.ClientIP()))

	var loginData struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	// 绑定 JSON 数据到结构体
	if err := c.ShouldBindJSON(&loginData); err != nil {
		logger.Warn("登录请求参数错误", clog.Err(err), clog.String("ip", c.ClientIP()))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数不正确"})
		return
	}

	// 获取用户信息
	var user model.User
	if err := utils.GetDB().Where("username = ?", loginData.Username).First(&user).Error; err != nil {
		logger.Warn("登录失败：用户不存在", clog.String("username", loginData.Username), clog.String("ip", c.ClientIP()))
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "用户名或密码错误"})
		return
	}

	// 验证密码
	if !utils.CheckPasswordHash(loginData.Password, user.Password) {
		logger.Warn("登录失败：密码错误", clog.String("username", user.Username), clog.String("ip", c.ClientIP()))
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "用户名或密码错误"})
		return
	}

	// 生成JWT令牌
	token, err := utils.GenerateToken(user.Username, string(user.Role), time.Hour*24)
	if err != nil {
		logger.Error("生成令牌失败", clog.Err(err), clog.String("username", user.Username))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	logger.Info("用户登录成功",
		clog.String("username", user.Username),
		clog.String("role", string(user.Role)),
		clog.String("ip", c.ClientIP()))

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "登录成功",
		"data": gin.H{
			"token": token,
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
				"email":    user.Email,
				"role":     user.Role,
			},
		},
	})
}

// Register 用户注册
func Register(c *gin.Context) {
	logger := clog.Module("user")
	logger.Info("处理注册请求", clog.String("ip", c.ClientIP()))

	var userData struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
	}

	// 绑定 JSON 数据到结构体
	if err := c.ShouldBindJSON(&userData); err != nil {
		logger.Warn("注册请求参数错误", clog.Err(err), clog.String("ip", c.ClientIP()))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数不正确"})
		return
	}

	// 检查用户名是否已存在
	var count int64
	utils.GetDB().Model(&model.User{}).Where("username = ?", userData.Username).Count(&count)
	if count > 0 {
		logger.Warn("注册失败：用户名已存在", clog.String("username", userData.Username), clog.String("ip", c.ClientIP()))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户名已存在"})
		return
	}

	// 检查邮箱是否已存在
	utils.GetDB().Model(&model.User{}).Where("email = ?", userData.Email).Count(&count)
	if count > 0 {
		logger.Warn("注册失败：邮箱已被使用", clog.String("email", userData.Email), clog.String("ip", c.ClientIP()))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "邮箱已被使用"})
		return
	}

	// 加密密码
	hashedPassword, err := utils.HashPassword(userData.Password)
	if err != nil {
		logger.Error("密码加密失败", clog.Err(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	// 创建用户
	user := model.User{
		Username: userData.Username,
		Password: hashedPassword,
		Email:    userData.Email,
		Role:     model.RoleGuest, // 默认为游客角色
	}

	if err := utils.GetDB().Create(&user).Error; err != nil {
		logger.Error("创建用户失败", clog.Err(err), clog.String("username", userData.Username))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	logger.Info("用户注册成功",
		clog.String("username", user.Username),
		clog.String("email", user.Email),
		clog.String("ip", c.ClientIP()))

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "注册成功",
		"data": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
		},
	})
}

// Logout 用户登出
func Logout(c *gin.Context) {
	logger := clog.Module("user")
	username := c.GetString("Username")

	token := c.GetHeader("Authorization")
	utils.RevokeToken(token, time.Hour*24) // 撤销令牌，保持在黑名单中 24 小时

	logger.Info("用户登出成功",
		clog.String("username", username),
		clog.String("ip", c.ClientIP()))

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "登出成功"})
}

// GetUsers 获取用户列表（仅管理员可用）
func GetUsers(c *gin.Context) {
	logger := clog.Module("user")
	logger.Info("获取用户列表请求", clog.String("requester", c.GetString("Username")))

	// 检查权限
	role, exists := c.Get("Role")
	if !exists || role.(string) != string(model.RoleAdmin) {
		logger.Warn("非管理员尝试获取用户列表",
			clog.String("username", c.GetString("Username")),
			clog.String("role", role.(string)),
			clog.String("ip", c.ClientIP()))
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "权限不足"})
		return
	}

	var users []model.User
	if err := utils.GetDB().Find(&users).Error; err != nil {
		logger.Error("查询用户列表失败", clog.Err(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	var userList []gin.H
	for _, user := range users {
		userList = append(userList, gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"role":      user.Role,
			"createdAt": user.CreatedAt,
		})
	}

	logger.Info("获取用户列表成功",
		clog.String("admin", c.GetString("Username")),
		clog.Int("count", len(users)))

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取成功",
		"data":    userList,
	})
}

// GetUserByID 通过ID获取用户信息（仅管理员或本人可查看）
func GetUserByID(c *gin.Context) {
	logger := clog.Module("user")
	userID := c.Param("id")
	currentUsername := c.GetString("Username")
	role := c.GetString("Role")

	logger.Info("获取用户详情请求",
		clog.String("target_id", userID),
		clog.String("requester", currentUsername))

	var user model.User
	if err := utils.GetDB().First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("查询的用户不存在", clog.String("user_id", userID))
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		} else {
			logger.Error("查询用户详情失败", clog.Err(err), clog.String("user_id", userID))
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		}
		return
	}

	// 检查权限：只有管理员或用户本人可以查看详细信息
	if role != string(model.RoleAdmin) && currentUsername != user.Username {
		logger.Warn("用户尝试访问其他用户的详情",
			clog.String("requester", currentUsername),
			clog.String("target", user.Username))
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "权限不足"})
		return
	}

	logger.Info("获取用户详情成功",
		clog.String("requester", currentUsername),
		clog.String("target", user.Username))

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取成功",
		"data": gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"role":      user.Role,
			"createdAt": user.CreatedAt,
		},
	})
}

// ChangePassword 修改密码（用户本人或管理员可操作）
func ChangePassword(c *gin.Context) {
	logger := clog.Module("user")
	logger.Info("修改密码请求", clog.String("requester", c.GetString("Username")))

	var passwordData struct {
		UserID      int    `json:"user_id,omitempty"`
		OldPassword string `json:"old_password,omitempty"`
		NewPassword string `json:"new_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&passwordData); err != nil {
		logger.Warn("修改密码参数错误", clog.Err(err), clog.String("requester", c.GetString("Username")))
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数不正确"})
		return
	}

	currentUsername := c.GetString("Username")
	role := c.GetString("Role")

	var user model.User
	var err error

	// 如果是管理员且指定了用户ID，则修改指定用户的密码
	if role == string(model.RoleAdmin) && passwordData.UserID > 0 {
		err = utils.GetDB().First(&user, passwordData.UserID).Error
	} else {
		// 否则修改当前登录用户的密码
		err = utils.GetDB().Where("username = ?", currentUsername).First(&user).Error
	}

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warn("修改密码的用户不存在", clog.Int("user_id", passwordData.UserID))
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
		} else {
			logger.Error("查询用户失败", clog.Err(err))
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		}
		return
	}

	// 如果不是管理员，需要验证旧密码
	if role != string(model.RoleAdmin) {
		if passwordData.OldPassword == "" {
			logger.Warn("修改密码未提供旧密码", clog.String("username", user.Username))
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "必须提供旧密码"})
			return
		}

		if !utils.CheckPasswordHash(passwordData.OldPassword, user.Password) {
			logger.Warn("修改密码旧密码验证失败", clog.String("username", user.Username))
			c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "旧密码不正确"})
			return
		}
	}

	// 更新密码
	hashedPassword, err := utils.HashPassword(passwordData.NewPassword)
	if err != nil {
		logger.Error("密码加密失败", clog.Err(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	user.Password = hashedPassword
	if err := utils.GetDB().Save(&user).Error; err != nil {
		logger.Error("保存新密码失败", clog.Err(err), clog.String("username", user.Username))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	logger.Info("密码修改成功",
		clog.String("target", user.Username),
		clog.String("requester", c.GetString("Username")))

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "密码修改成功",
	})
}

// GetCurrentUser 获取当前登录用户信息
func GetCurrentUser(c *gin.Context) {
	logger := clog.Module("user")
	username := c.GetString("Username")

	logger.Info("获取当前用户信息", clog.String("username", username))

	var user model.User
	if err := utils.GetDB().Where("username = ?", username).First(&user).Error; err != nil {
		logger.Error("获取当前用户信息失败", clog.Err(err), clog.String("username", username))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取成功",
		"data": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	})
}
