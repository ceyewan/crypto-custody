package handler

import (
	"errors"
	"net/http"
	"online-server/model"
	"online-server/service"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Login 用户登录
func Login(c *gin.Context) {
	var loginData struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	// 绑定 JSON 数据到结构体
	if err := c.ShouldBindJSON(&loginData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数不正确"})
		return
	}

	// 调用用户服务处理登录
	userService, err := service.GetUserServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	user, token, err := userService.Login(loginData.Username, loginData.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": err.Error()})
		return
	}

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
	var userData struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
	}

	// 绑定 JSON 数据到结构体
	if err := c.ShouldBindJSON(&userData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数不正确"})
		return
	}

	// 调用用户服务处理注册
	userService, err := service.GetUserServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	user, err := userService.Register(userData.Username, userData.Password, userData.Email)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

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
	token := c.GetHeader("Authorization")

	userService, err := service.GetUserServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	userService.Logout(token)

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "登出成功"})
}

// GetUsers 获取用户列表（仅管理员可用）
func GetUsers(c *gin.Context) {
	// 检查权限
	role, exists := c.Get("Role")
	if !exists || role.(string) != string(model.RoleAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "权限不足"})
		return
	}

	userService, err := service.GetUserServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	users, err := userService.GetUsers()
	if err != nil {
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

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取用户列表成功",
		"data":    userList,
	})
}

// GetUserByID 根据ID获取用户信息
func GetUserByID(c *gin.Context) {
	// 获取用户ID参数
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
		return
	}

	userService, err := service.GetUserServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	user, err := userService.GetUserByID(uint(userID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "用户不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取用户信息成功",
		"data": gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"role":      user.Role,
			"createdAt": user.CreatedAt,
		},
	})
}

// ChangePassword 修改密码
func ChangePassword(c *gin.Context) {
	var passwordData struct {
		OldPassword string `json:"oldPassword" binding:"required"`
		NewPassword string `json:"newPassword" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&passwordData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数不正确"})
		return
	}

	// 获取当前用户ID
	userID, exists := c.Get("UserID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未授权"})
		return
	}

	userService, err := service.GetUserServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	err = userService.ChangePassword(uint(userID.(uint)), passwordData.OldPassword, passwordData.NewPassword)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "密码修改成功",
	})
}

// GetCurrentUser 获取当前登录用户信息
func GetCurrentUser(c *gin.Context) {
	username := c.GetString("Username")

	userService, err := service.GetUserServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	user, err := userService.GetUserByUsername(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "获取当前用户信息成功",
		"data": gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"role":      user.Role,
			"createdAt": user.CreatedAt,
		},
	})
}

// UpdateUserRole 更新用户角色（仅管理员可用）
func UpdateUserRole(c *gin.Context) {
	// 检查权限
	role, exists := c.Get("Role")
	if !exists || role.(string) != string(model.RoleAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "权限不足"})
		return
	}

	// 获取用户ID参数
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
		return
	}

	var roleData struct {
		Role string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&roleData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请求参数不正确"})
		return
	}

	// 验证角色值是否有效
	var newRole model.Role
	switch model.Role(roleData.Role) {
	case model.RoleAdmin, model.RoleOfficer, model.RoleGuest:
		newRole = model.Role(roleData.Role)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的角色值"})
		return
	}

	userService, err := service.GetUserServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	err = userService.UpdateUserRole(uint(userID), newRole)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新角色失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "用户角色更新成功",
	})
}

// DeleteUser 删除用户（仅管理员可用）
func DeleteUser(c *gin.Context) {
	// 检查权限
	role, exists := c.Get("Role")
	if !exists || role.(string) != string(model.RoleAdmin) {
		c.JSON(http.StatusForbidden, gin.H{"code": 403, "message": "权限不足"})
		return
	}

	// 获取用户ID参数
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的用户ID"})
		return
	}

	userService, err := service.GetUserServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	err = userService.DeleteUser(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除用户失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "用户删除成功",
	})
}
