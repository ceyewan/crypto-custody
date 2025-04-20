package handler

import (
	"net/http"
	"offline-server/storage/model"
	"offline-server/tools"
	"offline-server/web/service"
	"time"

	"github.com/gin-gonic/gin"
)

// 登录请求结构
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// 注册请求结构
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

// 角色更新请求结构
type UpdateRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

// Login 处理用户登录请求
func Login(c *gin.Context) {
	var req LoginRequest
	if !bindJSON(c, &req) {
		return
	}

	// 调用服务层进行用户登录验证
	user, err := service.LoginUser(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 生成JWT令牌
	token, err := tools.GenerateToken(user.Username, user.Role, time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	})
}

// Register 处理用户注册请求
func Register(c *gin.Context) {
	var req RegisterRequest
	if !bindJSON(c, &req) {
		return
	}

	// 调用服务层创建新用户
	user, err := service.RegisterUser(req.Username, req.Password, req.Email)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "注册成功",
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		},
	})
}

// CheckAuth 验证用户认证状态
func CheckAuth(c *gin.Context) {
	// 从上下文中获取用户信息（由中间件设置）
	userName, exists := c.Get("userName")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "认证有效",
		"user":   userName,
	})
}

// Logout 处理用户登出
func Logout(c *gin.Context) {
	// JWT是无状态的，服务端不需要做任何操作
	// 客户端需要清除令牌
	c.JSON(http.StatusOK, gin.H{
		"message": "登出成功",
	})
}

// ListUsers 获取所有用户列表（仅限管理员）
func ListUsers(c *gin.Context) {
	// 调用服务层获取用户列表
	users, err := service.GetAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户列表失败"})
		return
	}

	var usersResponse []gin.H
	for _, user := range users {
		usersResponse = append(usersResponse, gin.H{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"role":     user.Role,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":  200,
		"users": usersResponse,
	})
}

// UpdateUserRole 更新用户角色
func UpdateUserRole(c *gin.Context) {
	// 获取用户ID参数
	userName := c.Param("id")

	// 获取请求体中的角色信息
	var req UpdateRoleRequest
	if !bindJSON(c, &req) {
		return
	}

	// 验证角色是否有效
	if !isValidRole(req.Role) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的角色"})
		return
	}

	// 调用服务层更新用户角色
	err := service.UpdateUserRole(userName, req.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新用户角色失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "用户角色更新成功",
	})
}

// isValidRole 检查角色是否有效
func isValidRole(role string) bool {
	validRoles := []string{
		string(model.Admin),
		string(model.Coordinator),
		string(model.Participant),
		string(model.Guest),
	}

	for _, validRole := range validRoles {
		if role == validRole {
			return true
		}
	}
	return false
}
