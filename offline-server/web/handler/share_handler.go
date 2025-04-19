// Package handler 提供 Web API 的请求处理函数
package handler

import (
	"fmt"
	"net/http"
	"offline-server/storage"
	"offline-server/web/service"

	"github.com/gin-gonic/gin"
)

// GetUserShares 获取用户所有密钥分享
func GetUserShares(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":  400,
			"error": "无法获取用户ID",
		})
		return
	}

	// 获取所有密钥分享
	shareStorage := storage.GetShareStorage()
	shares, err := shareStorage.GetUserShares(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":  500,
			"error": fmt.Sprintf("获取密钥分享失败: %v", err),
		})
		return
	}

	// 构造响应
	result := make([]gin.H, 0, len(shares))
	for keyID, shareJSON := range shares {
		result = append(result, gin.H{
			"key_id":     keyID,
			"user_id":    userID,
			"share_data": shareJSON,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":   200,
		"shares": result,
	})
}

// GetUserShare 获取用户特定密钥分享
func GetUserShare(c *gin.Context) {
	userID := c.GetString("userID")
	keyID := c.Param("keyID")

	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":  400,
			"error": "无法获取用户ID",
		})
		return
	}

	if keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":  400,
			"error": "缺少密钥ID",
		})
		return
	}

	// 获取特定密钥分享
	shareStorage := storage.GetShareStorage()
	shareJSON, err := shareStorage.GetUserShare(userID, keyID)
	if err != nil {
		if err == storage.ErrSessionNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"code":  404,
				"error": "密钥分享不存在",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":  500,
			"error": fmt.Sprintf("获取密钥分享失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"share": gin.H{
			"key_id":     keyID,
			"user_id":    userID,
			"share_data": shareJSON,
		},
	})
}

// GetSessionShares 获取特定会话的密钥分享
func GetSessionShares(c *gin.Context) {
	// 获取会话ID参数
	sessionKey := c.Param("session")
	if sessionKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未提供会话ID"})
		return
	}

	// 获取当前用户
	userName, exists := c.Get("userName")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
		return
	}

	// 获取用户角色
	role, exists := c.Get("role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
		return
	}

	// 非管理员只能查看自己的分享
	if role.(string) != "admin" {
		// 调用服务获取用户的特定分享
		share, err := service.GetUserShare(userName.(string), sessionKey)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "未找到分享数据或无权访问"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":  200,
			"share": share,
		})
		return
	}

	// 管理员可以查看所有用户的分享
	shares, err := service.GetSessionShares(sessionKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到分享数据"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":   200,
		"shares": shares,
	})
}

// GetSharesByAccount 通过账户地址获取分享
func GetSharesByAccount(c *gin.Context) {
	// 获取账户地址参数
	accountAddr := c.Param("addr")
	if accountAddr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未提供账户地址"})
		return
	}

	// 获取当前用户
	userName, exists := c.Get("userName")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
		return
	}

	// 获取用户角色
	role, exists := c.Get("role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
		return
	}

	// 调用服务获取账户相关的密钥生成会话
	session, err := service.GetKeyGenSessionByAccount(accountAddr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到账户信息"})
		return
	}

	// 非管理员只能查看自己参与的会话相关的分享
	if role.(string) != "admin" {
		isParticipant := false
		for _, p := range session.Participants {
			if p == userName.(string) {
				isParticipant = true
				break
			}
		}

		if !isParticipant {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权查看此账户的分享信息"})
			return
		}
	}

	// 调用服务获取账户相关的所有分享
	shares, err := service.GetSharesByAccount(accountAddr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到分享数据"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":   200,
		"shares": shares,
	})
}
