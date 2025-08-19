package controllers

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"offline-client-wails/clog"

	"github.com/gin-gonic/gin"

	"offline-client-wails/config"
	"offline-client-wails/models"
	"offline-client-wails/services"
)

var (
	cfg             *config.Config
	securityService *services.SecurityService
	mpcService      *services.MPCService
)

var initOnce sync.Once

// Init 初始化控制器
func Init() error {
	var initErr error

	initOnce.Do(func() {
		var err error

		// 加载配置
		cfg, err = config.LoadConfig()
		if err != nil {
			clog.Error("加载配置失败", clog.String("error", err.Error()))
			initErr = err
			return
		}

		// 创建安全芯片服务
		securityService, err = services.NewSecurityService(cfg)
		if err != nil {
			clog.Error("创建安全芯片服务失败", clog.String("error", err.Error()))
			initErr = err
			return
		}
		clog.Info("安全芯片服务初始化成功")

		// 创建MPC服务
		mpcService = services.NewMPCService(cfg, securityService)
		clog.Info("MPC控制器初始化成功")
	})

	return initErr
}

// Shutdown 关闭所有控制器相关资源
// 包括安全芯片服务在内的所有资源都会被正确释放
func Shutdown() {
	clog.Info("控制器资源清理开始")

	// 关闭安全芯片服务
	if securityService != nil {
		clog.Info("关闭安全芯片服务")
		securityService.Close()
		securityService = nil
	}

	// 清理其他资源
	mpcService = nil
	cfg = nil

	clog.Info("控制器资源清理完成")
}

// KeyGeneration 处理密钥生成请求
func KeyGeneration(c *gin.Context) {
	// 确保服务已初始化
	if cfg == nil || securityService == nil || mpcService == nil {
		clog.Info("服务未初始化，尝试初始化")
		if err := Init(); err != nil {
			clog.Error("服务初始化失败", clog.String("error", err.Error()))
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
		clog.Warn("请求参数解析失败",
			clog.Err(err),
			clog.String("client_ip", c.ClientIP()))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	clog.Info("接收到密钥生成请求",
		clog.Int("threshold", req.Threshold),
		clog.Int("parties", req.Parties),
		clog.Int("index", req.Index),
		clog.String("filename", req.Filename),
		clog.String("username", req.UserName))

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
		clog.Error("密钥生成失败",
			clog.String("error", err.Error()),
			clog.String("username", req.UserName))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "密钥生成失败: " + err.Error(),
		})
		return
	}

	clog.Info("密钥生成成功",
		clog.String("address", address),
		clog.String("username", req.UserName),
		clog.Int("encrypted_key_length", len(encryptedKey)))

	// 确保地址格式正确（添加0x前缀如果没有）
	if !strings.HasPrefix(address, "0x") {
		address = "0x" + address
		clog.Debug("地址格式规范化", clog.String("address", address))
	}

	// 转换加密密钥为base64字符串
	encryptedKeyBase64 := base64.StdEncoding.EncodeToString(encryptedKey)

	// 返回响应
	c.JSON(http.StatusOK, models.KeyGenResponse{
		Success:      true,
		UserName:     req.UserName,
		Address:      address,
		EncryptedKey: encryptedKeyBase64,
	})
}

// SignMessage 处理签名请求
func SignMessage(c *gin.Context) {
	// 确保服务已初始化
	if cfg == nil || securityService == nil || mpcService == nil {
		clog.Info("服务未初始化，尝试初始化")
		if err := Init(); err != nil {
			clog.Error("服务初始化失败", clog.String("error", err.Error()))
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
		clog.Warn("请求参数解析失败",
			clog.String("error", err.Error()),
			clog.String("client_ip", c.ClientIP()))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 验证地址格式并标准化
	if !strings.HasPrefix(req.Address, "0x") {
		req.Address = "0x" + req.Address
		clog.Debug("地址格式规范化", clog.String("address", req.Address))
	}

	// 解码base64加密密钥
	encryptedKey, err := base64.StdEncoding.DecodeString(req.EncryptedKey)
	if err != nil {
		clog.Error("加密密钥解码失败",
			clog.String("error", err.Error()),
			clog.String("username", req.UserName))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "加密密钥格式错误: " + err.Error(),
		})
		return
	}

	// 解码签名（DER格式）
	signature, err := base64.StdEncoding.DecodeString(req.Signature)
	if err != nil {
		clog.Error("签名解码失败",
			clog.String("error", err.Error()),
			clog.String("username", req.UserName))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "签名格式错误: " + err.Error(),
		})
		return
	}

	clog.Info("接收到签名请求",
		clog.String("parties", req.Parties),
		clog.String("data", req.Data),
		clog.String("filename", req.Filename),
		clog.String("username", req.UserName),
		clog.String("address", req.Address))
	clog.Debug("签名请求详情",
		clog.String("encrypted_key_length", formatByteSize(int64(len(encryptedKey)))),
		clog.String("signature_length", formatByteSize(int64(len(signature)))))

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
		clog.Error("签名失败",
			clog.String("error", err.Error()),
			clog.String("username", req.UserName),
			clog.String("address", req.Address))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "签名失败: " + err.Error(),
		})
		return
	}

	// 确保以太坊签名格式（0x前缀）
	if !strings.HasPrefix(signatureResult, "0x") {
		signatureResult = "0x" + signatureResult
		clog.Debug("签名格式规范化", clog.String("signature", signatureResult))
	}

	clog.Info("签名成功",
		clog.String("username", req.UserName),
		clog.String("address", req.Address))
	clog.Debug("签名结果", clog.String("signature", signatureResult))

	// 返回响应
	c.JSON(http.StatusOK, models.SignResponse{
		Success:   true,
		Signature: signatureResult,
	})
}

// GetCPLC 获取安全芯片的CPLC信息
func GetCPLC(c *gin.Context) {
	// 确保服务已初始化
	if cfg == nil || securityService == nil || mpcService == nil {
		clog.Info("服务未初始化，尝试初始化")
		if err := Init(); err != nil {
			clog.Error("服务初始化失败", clog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "服务初始化失败: " + err.Error(),
			})
			return
		}
	}

	clog.Info("接收到获取CPLC信息请求", clog.String("client_ip", c.ClientIP()))

	// 调用服务获取CPLC信息
	cplcData, err := securityService.GetCPLC()
	if err != nil {
		clog.Error("获取CPLC信息失败", clog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "获取CPLC信息失败: " + err.Error(),
		})
		return
	}

	// 转换为十六进制字符串
	cplcHex := fmt.Sprintf("%X", cplcData)
	clog.Info("获取CPLC信息成功", clog.String("cplc", cplcHex))

	// 返回响应
	c.JSON(http.StatusOK, models.GetCPLCResponse{
		Success: true,
		CPIC:    cplcHex,
	})
}

// DeleteMessage 删除用户数据
func DeleteMessage(c *gin.Context) {
	// 确保服务已初始化
	if cfg == nil || securityService == nil || mpcService == nil {
		clog.Info("服务未初始化，尝试初始化")
		if err := Init(); err != nil {
			clog.Error("服务初始化失败", clog.String("error", err.Error()))
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"message": "服务初始化失败: " + err.Error(),
			})
			return
		}
	}

	// 解析请求
	var req models.DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		clog.Warn("请求参数解析失败",
			clog.String("error", err.Error()),
			clog.String("client_ip", c.ClientIP()))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "请求参数错误: " + err.Error(),
		})
		return
	}

	// 验证地址格式并标准化
	if !strings.HasPrefix(req.Address, "0x") {
		req.Address = "0x" + req.Address
		clog.Debug("地址格式规范化", clog.String("address", req.Address))
	}

	// 解码签名（DER格式）
	signature, err := base64.StdEncoding.DecodeString(req.Signature)
	if err != nil {
		clog.Error("签名解码失败",
			clog.String("error", err.Error()),
			clog.String("username", req.UserName))
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "签名格式错误: " + err.Error(),
		})
		return
	}

	clog.Info("接收到删除数据请求",
		clog.String("username", req.UserName),
		clog.String("address", req.Address))
	clog.Debug("删除请求详情",
		clog.String("signature_length", formatByteSize(int64(len(signature)))))

	// 删除数据
	err = securityService.DeleteData(req.UserName, req.Address, signature)
	if err != nil {
		clog.Error("删除数据失败",
			clog.String("error", err.Error()),
			clog.String("username", req.UserName),
			clog.String("address", req.Address))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "删除数据失败: " + err.Error(),
		})
		return
	}

	clog.Info("删除数据成功",
		clog.String("username", req.UserName),
		clog.String("address", req.Address))

	// 返回响应
	c.JSON(http.StatusOK, models.DeleteResponse{
		Success: true,
		Address: req.Address,
	})
}

// formatByteSize 格式化字节大小为人类可读的字符串
func formatByteSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
