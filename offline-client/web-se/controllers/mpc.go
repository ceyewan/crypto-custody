package controllers

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"web-se/config"
	"web-se/models"
	"web-se/services"
	"web-se/utils"
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
		utils.LogError("加载配置失败", utils.Error(err))
		return err
	}

	// 创建安全芯片服务
	securityService, err = services.NewSecurityService(cfg)
	if err != nil {
		utils.LogError("创建安全芯片服务失败", utils.Error(err))
		return err
	}

	// 创建MPC服务
	mpcService = services.NewMPCService(cfg, securityService)
	utils.LogInfo("MPC控制器初始化成功")

	return nil
}

// KeyGeneration 处理密钥生成请求
func KeyGeneration(c *gin.Context) {
	// 确保服务已初始化
	if cfg == nil || securityService == nil || mpcService == nil {
		utils.LogInfo("服务未初始化，尝试初始化")
		if err := Init(); err != nil {
			utils.LogError("服务初始化失败", utils.Error(err))
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
		utils.LogWarn("请求参数解析失败",
			utils.Error(err),
			utils.String("client_ip", c.ClientIP()))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	utils.LogInfo("接收到密钥生成请求",
		utils.Int("threshold", req.Threshold),
		utils.Int("parties", req.Parties),
		utils.Int("index", req.Index),
		utils.String("filename", req.Filename),
		utils.String("username", req.UserName))

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
		utils.LogError("密钥生成失败",
			utils.Error(err),
			utils.String("username", req.UserName))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "密钥生成失败: " + err.Error(),
		})
		return
	}

	// 确保地址格式正确（添加0x前缀如果没有）
	if !strings.HasPrefix(address, "0x") {
		address = "0x" + address
		utils.LogDebug("地址格式规范化", utils.String("address", address))
	}

	// 转换加密密钥为base64字符串
	encryptedKeyBase64 := base64.StdEncoding.EncodeToString(encryptedKey)

	utils.LogInfo("密钥生成成功",
		utils.String("address", address),
		utils.String("username", req.UserName))
	utils.LogDebug("密钥详情",
		utils.String("encrypted_key_length", utils.FormatByteSize(int64(len(encryptedKey)))),
		utils.String("address", address))

	// 返回响应
	c.JSON(http.StatusOK, models.KeyGenResponse{
		Success:      true,
		Address:      address,
		EncryptedKey: encryptedKeyBase64,
	})
}

// SignMessage 处理签名请求
func SignMessage(c *gin.Context) {
	// 确保服务已初始化
	if cfg == nil || securityService == nil || mpcService == nil {
		utils.LogInfo("服务未初始化，尝试初始化")
		if err := Init(); err != nil {
			utils.LogError("服务初始化失败", utils.Error(err))
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
		utils.LogWarn("请求参数解析失败",
			utils.Error(err),
			utils.String("client_ip", c.ClientIP()))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 验证地址格式并标准化
	if !strings.HasPrefix(req.Address, "0x") {
		req.Address = "0x" + req.Address
		utils.LogDebug("地址格式规范化", utils.String("address", req.Address))
	}

	// 解码base64加密密钥
	encryptedKey, err := base64.StdEncoding.DecodeString(req.EncryptedKey)
	if err != nil {
		utils.LogError("加密密钥解码失败",
			utils.Error(err),
			utils.String("username", req.UserName))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "加密密钥格式错误: " + err.Error(),
		})
		return
	}

	// 解码签名（DER格式）
	signature, err := base64.StdEncoding.DecodeString(req.Signature)
	if err != nil {
		utils.LogError("签名解码失败",
			utils.Error(err),
			utils.String("username", req.UserName))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "签名格式错误: " + err.Error(),
		})
		return
	}

	utils.LogInfo("接收到签名请求",
		utils.String("parties", req.Parties),
		utils.String("data", req.Data),
		utils.String("filename", req.Filename),
		utils.String("username", req.UserName),
		utils.String("address", req.Address))
	utils.LogDebug("签名请求详情",
		utils.String("encrypted_key_length", utils.FormatByteSize(int64(len(encryptedKey)))),
		utils.String("signature_length", utils.FormatByteSize(int64(len(signature)))))

	// 调用服务进行签名
	signatureResult, err := mpcService.SignMessage(
		c.Request.Context(),
		req.Parties,
		req.Data,
		req.Filename,
		req.UserName,
		req.Address,
		encryptedKey,
		signature,
	)
	if err != nil {
		utils.LogError("签名失败",
			utils.Error(err),
			utils.String("username", req.UserName),
			utils.String("address", req.Address))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "签名失败: " + err.Error(),
		})
		return
	}

	// 确保以太坊签名格式（0x前缀）
	if !strings.HasPrefix(signatureResult, "0x") {
		signatureResult = "0x" + signatureResult
		utils.LogDebug("签名格式规范化", utils.String("signature", signatureResult))
	}

	utils.LogInfo("签名成功",
		utils.String("username", req.UserName),
		utils.String("address", req.Address))
	utils.LogDebug("签名结果", utils.String("signature", signatureResult))

	// 返回响应
	c.JSON(http.StatusOK, models.SignResponse{
		Success:   true,
		Signature: signatureResult,
	})
}
