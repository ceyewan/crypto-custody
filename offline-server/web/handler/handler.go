package handler

import (
	"net/http"
	"strconv"
	"time"

	"offline-server/tools"
	"offline-server/web/service"

	"github.com/gin-gonic/gin"
)

// 全局服务实例
var (
	authService = service.NewAuthService()
)

// 处理请求参数绑定的通用函数
func bindJSON(c *gin.Context, req interface{}) bool {
	if err := c.BindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return false
	}
	return true
}

// Login 处理用户登录请求
func Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if !bindJSON(c, &req) {
		return
	}

	// 调用服务层进行登录验证
	user, err := authService.Login(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 生成JWT token
	token, err := tools.GenerateToken(int(user.ID), user.Role, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  user,
	})
}

// Register 处理用户注册请求
func Register(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	if !bindJSON(c, &req) {
		return
	}

	user, err := authService.Register(req.Username, req.Password, req.Email)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "注册成功",
		"user":    user,
	})
}

// ListUsers 获取用户列表
func ListUsers(c *gin.Context) {
	users, err := authService.ListUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取用户列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":  http.StatusOK,
		"users": users,
	})
}

// UpdateUserRole 更新用户角色
func UpdateUserRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	var req struct {
		Role string `json:"role"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 更新用户角色
	if err := authService.UpdateUserRole(uint(id), req.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新用户角色失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": http.StatusOK,
		"msg":  "用户角色更新成功",
	})
}

// Logout 处理用户登出请求
func Logout(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少认证令牌"})
		return
	}

	// 撤销令牌
	tools.RevokeToken(token, 24*time.Hour)
	c.JSON(http.StatusOK, gin.H{"message": "登出成功"})
}

// CheckAuth 验证用户认证状态
func CheckAuth(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证令牌"})
		return
	}

	// 验证令牌
	_, _, err := tools.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的认证令牌"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "认证有效"})
}
