package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"web-se/config"
	"web-se/models"
	"web-se/services"
)

var (
	cfg             *config.Config
	securityService *services.SecurityService
	mpcService      *services.MPCService
)

// Init 初始化控制器
func Init() error {
	var err error

	// 加载配置
	cfg, err = config.LoadConfig()
	if err != nil {
		return err
	}

	// 创建安全芯片服务
	securityService, err = services.NewSecurityService(cfg)
	if err != nil {
		return err
	}

	// 创建MPC服务
	mpcService = services.NewMPCService(cfg, securityService)

	return nil
}

// KeyGeneration 处理密钥生成请求
func KeyGeneration(c *gin.Context) {
	// 确保服务已初始化
	if cfg == nil || securityService == nil || mpcService == nil {
		if err := Init(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "服务初始化失败: " + err.Error(),
			})
			return
		}
	}

	// 解析请求
	var req models.KeyGenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 调用服务生成密钥
	address, encryptedKey, err := mpcService.KeyGeneration(
		c.Request.Context(),
		req.Threshold,
		req.Parties,
		req.Index,
		req.Filename,
		req.UserName,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "密钥生成失败: " + err.Error(),
		})
		return
	}

	// 返回响应
	c.JSON(http.StatusOK, models.KeyGenResponse{
		Success:      true,
		Address:      address,
		EncryptedKey: encryptedKey,
	})
}

// SignMessage 处理签名请求
func SignMessage(c *gin.Context) {
	// 确保服务已初始化
	if cfg == nil || securityService == nil || mpcService == nil {
		if err := Init(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "服务初始化失败: " + err.Error(),
			})
			return
		}
	}

	// 解析请求
	var req models.SignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 调用服务进行签名
	signature, err := mpcService.SignMessage(
		c.Request.Context(),
		req.Parties,
		req.Data,
		req.Filename,
		req.UserName,
		req.Address,
		req.EncryptedKey,
		req.Signature,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "签名失败: " + err.Error(),
		})
		return
	}

	// 返回响应
	c.JSON(http.StatusOK, models.SignResponse{
		Success:   true,
		Signature: signature,
	})
}
