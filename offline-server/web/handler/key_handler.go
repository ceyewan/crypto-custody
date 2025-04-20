// Package handler 提供 Web API 的请求处理函数
package handler

import (
	"fmt"
	"net/http"
	"offline-server/storage/model"
	"offline-server/web/service"

	"github.com/gin-gonic/gin"
)

// 创建密钥生成请求结构
type GenerateKeyRequest struct {
	Threshold    int      `json:"threshold" binding:"required,min=1"`
	TotalParts   int      `json:"total_parts" binding:"required,min=2"`
	Participants []string `json:"participants" binding:"required"`
}

// GenerateKey 处理密钥生成请求
func GenerateKey(c *gin.Context) {
	var req GenerateKeyRequest
	if !bindJSON(c, &req) {
		return
	}

	// 参数验证
	if req.Threshold > len(req.Participants) || req.TotalParts != len(req.Participants) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "阈值不能大于参与者数量且总分片数必须等于参与者数量",
		})
		return
	}

	// 获取发起者信息
	initiator, exists := c.Get("userName")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权操作"})
		return
	}

	// 调用服务层创建密钥生成会话
	sessionKey, err := service.CreateKeyGenSession(
		initiator.(string),
		req.Threshold,
		req.TotalParts,
		req.Participants,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":        200,
		"session_key": sessionKey,
		"status":      "invited",
	})
}

// GetKeyGenSession 获取密钥生成会话详情
func GetKeyGenSession(c *gin.Context) {
	session, err := getSessionAndCheckPermission(c)
	if err != nil {
		return
	}

	// 构建响应
	response := gin.H{
		"session_key":  session.SessionKey,
		"initiator":    session.Initiator,
		"threshold":    session.Threshold,
		"total_parts":  session.TotalParts,
		"participants": session.Participants,
		"status":       session.Status,
		"account_addr": session.AccountAddr,
		"created_at":   session.CreatedAt,
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"session": response,
	})
}

// KeyGenStatus 获取密钥生成任务状态
func KeyGenStatus(c *gin.Context) {
	session, err := getSessionAndCheckPermission(c)
	if err != nil {
		return
	}

	// 构建简洁响应
	c.JSON(http.StatusOK, gin.H{
		"code":        200,
		"session_key": session.SessionKey,
		"type":        "keygen",
		"status":      session.Status,
	})
}

// 提取公共逻辑的辅助函数
func getSessionAndCheckPermission(c *gin.Context) (*model.KeyGenSession, error) {
	// 获取会话ID
	sessionKey := c.Param("id")
	if sessionKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未提供会话ID"})
		return nil, fmt.Errorf("missing session ID")
	}

	// 调用服务层获取会话详情
	session, err := service.GetKeyGenSession(sessionKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在或已过期"})
		return nil, fmt.Errorf("session not found")
	}

	// 检查用户权限
	userName, exists := c.Get("userName")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
		return nil, fmt.Errorf("unauthorized")
	}

	role, exists := c.Get("role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
		return nil, fmt.Errorf("unauthorized")
	}

	// 非管理员只能查看自己参与的会话
	if role.(string) != "admin" {
		isParticipant := false
		// 检查是否是发起者
		if session.Initiator == userName.(string) {
			isParticipant = true
		} else {
			// 检查是否在参与者列表中
			for _, participant := range session.Participants {
				if participant == userName.(string) {
					isParticipant = true
					break
				}
			}
		}

		if !isParticipant {
			c.JSON(http.StatusForbidden, gin.H{"error": "无权查看此会话"})
			return nil, fmt.Errorf("forbidden")
		}
	}

	return session, nil
}
