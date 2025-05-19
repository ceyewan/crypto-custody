package handler

import (
	"net/http"
	"online-server/service"
	"online-server/utils"

	"github.com/gin-gonic/gin"
)

// GetAccountByAddress 根据账户地址查询账户信息
//
// 通过URL参数获取地址，并返回对应的账户详情
//
// 路由: GET /api/accounts/address/:address
func GetAccountByAddress(c *gin.Context) {
	// 获取URL中的地址参数
	address := c.Param("address")
	if address == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "缺少地址参数",
		})
		return
	}

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "服务初始化失败",
			"error":   err.Error(),
		})
		return
	}

	// 查询账户信息
	account, err := accountService.GetAccountByAddress(address)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    404,
			"message": "查询账户失败",
			"error":   err.Error(),
		})
		return
	}

	// 返回账户信息
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "查询账户成功",
		"data":    account,
	})
}

// GetUserAccounts 获取当前登录用户导入的所有账户
//
// 需要JWT认证，从Token中获取用户名，返回该用户导入的所有账户列表
//
// 路由: GET /api/accounts
func GetUserAccounts(c *gin.Context) {
	// 从JWT中获取用户名
	claims, exists := c.Get("claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "无法获取用户信息",
		})
		return
	}

	userClaims, ok := claims.(*utils.Claims)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "用户信息类型错误",
		})
		return
	}

	username := userClaims.Username

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "服务初始化失败",
			"error":   err.Error(),
		})
		return
	}

	// 查询该用户导入的所有账户
	accounts, err := accountService.GetAccountsByImportedBy(username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询账户列表失败",
			"error":   err.Error(),
		})
		return
	}

	// 返回账户列表
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "查询账户列表成功",
		"data":    accounts,
	})
}

// GetAllAccounts 获取系统中的所有账户(仅管理员)
//
// 需要JWT认证，且用户必须具有管理员权限
//
// 路由: GET /api/accounts/all
func GetAllAccounts(c *gin.Context) {
	// 从JWT中获取用户信息和角色
	claims, exists := c.Get("claims")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "无法获取用户信息",
		})
		return
	}

	userClaims, ok := claims.(*utils.Claims)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "用户信息类型错误",
		})
		return
	}

	// 检查是否为管理员
	if !userClaims.IsAdmin {
		c.JSON(http.StatusForbidden, gin.H{
			"code":    403,
			"message": "权限不足，只有管理员可以访问所有账户信息",
		})
		return
	}

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "服务初始化失败",
			"error":   err.Error(),
		})
		return
	}

	// 查询所有账户
	accounts, err := accountService.GetAccounts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询所有账户失败",
			"error":   err.Error(),
		})
		return
	}

	// 返回所有账户信息
	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "查询所有账户成功",
		"data":    accounts,
		"total":   len(accounts),
	})
}
