package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"offline-server/storage"
	"offline-server/storage/model"
	"offline-server/tools"
	"offline-server/web/service"

	"github.com/gin-gonic/gin"
)

// 全局服务实例
var (
	authService   = service.NewAuthService()
	keyGenStorage = storage.GetKeyGenStorage()
	signStorage   = storage.GetSignStorage()
	shareStorage  = storage.GetShareStorage()
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

	// 生成JWT token - 使用用户名而非用户ID
	token, err := tools.GenerateToken(user.Username, user.Role, 24*time.Hour)
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

	// 生成密钥ID
	keyID := fmt.Sprintf("key_%s_%d", time.Now().Format("20060102150405"), time.Now().UnixNano()%10000)

	// 获取发起人ID
	initiatorID, _ := c.Get("userID")
	initiatorIDStr := fmt.Sprintf("%v", initiatorID)

	// 创建密钥生成会话
	err := keyGenStorage.CreateSession(keyID, initiatorIDStr, req.Threshold, len(req.Participants), req.Participants)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("创建密钥生成会话失败: %v", err)})
		return
	}

	// 更新会话状态为已邀请
	err = keyGenStorage.UpdateStatus(keyID, model.StatusInvited)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("更新会话状态失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":   http.StatusOK,
		"key_id": keyID,
		"status": string(model.StatusInvited),
	})
}

// CreateSignature 处理签名请求
func CreateSignature(c *gin.Context) {
	var req struct {
		KeyID        string   `json:"key_id"`
		Data         string   `json:"data"`
		Participants []string `json:"participants"`
		AccountAddr  string   `json:"account_addr"`
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

	// 获取发起人ID
	initiatorID, _ := c.Get("userID")
	initiatorIDStr := fmt.Sprintf("%v", initiatorID)

	// 创建签名会话
	err := signStorage.CreateSession(req.KeyID, initiatorIDStr, req.Data, req.AccountAddr, req.Participants)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("创建签名会话失败: %v", err)})
		return
	}

	// 更新会话状态为已邀请
	err = signStorage.UpdateStatus(req.KeyID, model.StatusInvited)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("更新会话状态失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    http.StatusOK,
		"sign_id": req.KeyID,
		"key_id":  req.KeyID,
		"status":  string(model.StatusInvited),
	})
}

// KeyStatus 获取密钥或签名任务状态
func KeyStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少ID参数"})
		return
	}

	// 尝试获取密钥生成会话
	keyGenSession, err := keyGenStorage.GetSession(id)
	if err == nil {
		// 返回密钥生成会话状态
		c.JSON(http.StatusOK, gin.H{
			"code":   http.StatusOK,
			"id":     id,
			"type":   "keygen",
			"status": string(keyGenSession.Status),
			"detail": keyGenSession,
		})
		return
	}

	// 尝试获取签名会话
	signSession, err := signStorage.GetSession(id)
	if err == nil {
		// 返回签名会话状态
		c.JSON(http.StatusOK, gin.H{
			"code":   http.StatusOK,
			"id":     id,
			"type":   "sign",
			"status": string(signSession.Status),
			"detail": signSession,
		})
		return
	}

	// 找不到指定ID的任务
	c.JSON(http.StatusNotFound, gin.H{"error": "找不到指定ID的任务"})
}

// GetUserShares 获取用户的密钥分享
func GetUserShares(c *gin.Context) {
	// 获取用户ID
	userID, _ := c.Get("userID")
	userIDStr := fmt.Sprintf("%v", userID)

	shares, err := shareStorage.GetUserShares(userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取用户密钥分享失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":   http.StatusOK,
		"shares": shares,
	})
}

// GetUserShare 获取用户的特定密钥分享
func GetUserShare(c *gin.Context) {
	keyID := c.Param("keyID")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少密钥ID参数"})
		return
	}

	// 获取用户ID
	userID, _ := c.Get("userID")
	userIDStr := fmt.Sprintf("%v", userID)

	share, err := shareStorage.GetUserShare(userIDStr, keyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("获取用户密钥分享失败: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":  http.StatusOK,
		"share": share,
	})
}
