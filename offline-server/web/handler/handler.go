package handler

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
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

// 密钥操作相关处理函数

// GenerateKey 处理密钥生成请求
func GenerateKey(c *gin.Context) {
	var req struct {
		Threshold    int      `json:"threshold"`
		Participants []string `json:"participants"`
	}

	if !bindJSON(c, &req) {
		return
	}

	// 验证参数
	if req.Threshold <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "阈值必须为正数"})
		return
	}

	if len(req.Participants) < req.Threshold {
		c.JSON(http.StatusBadRequest, gin.H{"error": "参与方数量必须不少于阈值"})
		return
	}

	// 生成随机的密钥ID
	keyID := fmt.Sprintf("key_%s_%d", time.Now().Format("20060102150405"), rand.Intn(10000))

	// 在实际系统中，这里应该将密钥生成任务保存到数据库

	c.JSON(http.StatusOK, gin.H{
		"code":   http.StatusOK,
		"key_id": keyID,
		"status": "pending",
	})
}

// CreateSignature 处理签名请求
func CreateSignature(c *gin.Context) {
	var req struct {
		KeyID        string   `json:"key_id"`
		Data         string   `json:"data"`
		Participants []string `json:"participants"`
	}

	if !bindJSON(c, &req) {
		return
	}

	// 验证参数
	if req.KeyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少密钥ID"})
		return
	}

	if req.Data == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少待签名数据"})
		return
	}

	if len(req.Participants) < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "至少需要一个参与方"})
		return
	}

	// 生成随机的签名ID
	signID := fmt.Sprintf("sign_%s_%d", time.Now().Format("20060102150405"), rand.Intn(10000))

	// 在实际系统中，这里应该将签名任务保存到数据库

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"sign_id": signID,
		"key_id":  req.KeyID,
		"status":  "pending",
	})
}

// KeyStatus 获取密钥或签名任务状态
func KeyStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少ID参数"})
		return
	}

	// 在实际系统中，这里应该从数据库查询任务状态
	// 现在我们返回一个模拟的状态

	var status string
	if strings.HasPrefix(id, "key_") {
		status = "generated"
	} else if strings.HasPrefix(id, "sign_") {
		status = "completed"
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "找不到指定ID的任务"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":   http.StatusOK,
		"id":     id,
		"status": status,
	})
}
