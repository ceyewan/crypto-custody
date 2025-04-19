// Package handler 提供 Web API 的请求处理函数
package handler

import (
	"fmt"
	"net/http"
	"offline-server/storage"
	"offline-server/storage/model"
	"offline-server/web/service"
	"time"

	"github.com/gin-gonic/gin"
)

// 签名会话创建请求结构
type CreateSignSessionRequest struct {
	KeyID        string   `json:"key_id" binding:"required"`       // 关联的密钥ID
	Data         string   `json:"data" binding:"required"`         // 需要签名的数据
	AccountAddr  string   `json:"account_addr" binding:"required"` // 账户地址
	Participants []string `json:"participants,omitempty"`          // 参与者列表(可选)，不提供则使用密钥生成时的参与者
}

// 签名请求结构
type CreateSignatureRequest struct {
	KeyID        string   `json:"key_id" binding:"required"`
	Data         string   `json:"data" binding:"required"`
	Participants []string `json:"participants" binding:"required"`
	AccountAddr  string   `json:"account_addr" binding:"required"`
}

// CreateSignSession 创建签名会话
func CreateSignSession(c *gin.Context) {
	var req CreateSignSessionRequest
	if !bindJSON(c, &req) {
		return
	}

	userID := c.GetString("userID")

	// 生成唯一的会话密钥（这里使用密钥ID作为会话密钥）
	sessionKey := fmt.Sprintf("sign_%s_%d", time.Now().Format("20060102150405"), time.Now().UnixNano()%10000)

	// 确定参与者列表
	participants := req.Participants

	// 如果未指定参与者，则使用密钥生成时的参与者
	if len(participants) == 0 {
		keyGenStorage := storage.GetKeyGenStorage()
		keyGenSession, err := keyGenStorage.GetSession(req.KeyID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":  400,
				"error": fmt.Sprintf("获取密钥会话失败: %v", err),
			})
			return
		}
		participants = keyGenSession.Participants
	}

	// 创建签名会话
	signStorage := storage.GetSignStorage()
	err := signStorage.CreateSession(
		sessionKey,
		userID,
		req.Data,
		req.AccountAddr,
		participants,
	)

	if err != nil {
		if err == storage.ErrSessionExists {
			c.JSON(http.StatusConflict, gin.H{
				"code":  409,
				"error": "签名会话已存在",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":  500,
			"error": fmt.Sprintf("创建签名会话失败: %v", err),
		})
		return
	}

	// 更新会话状态为已邀请
	err = signStorage.UpdateStatus(sessionKey, model.StatusInvited)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":  500,
			"error": fmt.Sprintf("更新会话状态失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":         200,
		"session_key":  sessionKey,
		"key_id":       req.KeyID,
		"account_addr": req.AccountAddr,
		"status":       "invited",
		"participants": participants,
	})
}

// CreateSignature 创建签名任务
func CreateSignature(c *gin.Context) {
	var req CreateSignatureRequest
	if !bindJSON(c, &req) {
		return
	}

	// 获取发起者信息
	initiator, exists := c.Get("userName")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权操作"})
		return
	}

	// 调用服务层创建签名会话
	sessionKey, err := service.CreateSignSession(
		initiator.(string),
		req.KeyID,
		req.Data,
		req.AccountAddr,
		req.Participants,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"sign_id": sessionKey,
		"key_id":  req.KeyID,
		"status":  "invited",
	})
}

// GetSignSession 获取签名会话信息
func GetSignSession(c *gin.Context) {
	// 获取会话ID
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未提供会话ID"})
		return
	}

	// 调用服务层获取会话详情
	session, err := service.GetSignSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在或已过期"})
		return
	}

	// 检查用户权限
	userName, exists := c.Get("userName")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
		return
	}

	role, exists := c.Get("role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
		return
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
			return
		}
	}

	// 构建响应
	response := gin.H{
		"session_key":  session.SessionKey,
		"initiator":    session.Initiator,
		"account_addr": session.AccountAddr,
		"participants": session.Participants,
		"status":       session.Status,
		"created_at":   session.CreatedAt,
		"signature":    session.Signature,
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"session": response,
	})
}

// SignStatus 获取签名任务状态
func SignStatus(c *gin.Context) {
	// 获取会话ID
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未提供会话ID"})
		return
	}

	// 调用服务层获取会话状态
	session, err := service.GetSignSession(sessionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在或已过期"})
		return
	}

	// 检查用户权限
	userName, exists := c.Get("userName")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
		return
	}

	role, exists := c.Get("role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
		return
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
			return
		}
	}

	// 构建简洁响应
	c.JSON(http.StatusOK, gin.H{
		"code":   200,
		"id":     session.SessionKey,
		"type":   "sign",
		"status": session.Status,
	})
}

// GetSignByAccount 根据账户地址查询签名会话
func GetSignByAccount(c *gin.Context) {
	// 获取账户地址参数
	accountAddr := c.Param("addr")
	if accountAddr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未提供账户地址"})
		return
	}

	// 调用服务层根据账户地址查询
	sessions, err := service.GetSignSessionsByAccount(accountAddr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	// 检查用户权限
	userName, exists := c.Get("userName")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
		return
	}

	role, exists := c.Get("role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
		return
	}

	// 构建响应
	var sessionsResponse []gin.H
	for _, session := range sessions {
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
				continue
			}
		}

		sessionsResponse = append(sessionsResponse, gin.H{
			"session_key":  session.SessionKey,
			"initiator":    session.Initiator,
			"account_addr": session.AccountAddr,
			"status":       session.Status,
			"created_at":   session.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":     200,
		"sessions": sessionsResponse,
	})
}

// GetParticipantsByAccount 获取账户参与者
func GetParticipantsByAccount(c *gin.Context) {
	// 获取账户地址参数
	accountAddr := c.Param("addr")
	if accountAddr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "未提供账户地址"})
		return
	}

	// 调用服务层获取该账户相关的密钥生成会话
	session, participants, err := service.GetParticipantsByAccount(accountAddr)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "找不到账户信息"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":         200,
		"key_id":       session.SessionKey,
		"account_addr": session.AccountAddr,
		"threshold":    session.Threshold,
		"participants": participants,
		"total_parts":  session.TotalParts,
		"created_at":   session.CreatedAt,
	})
}
