package handler

import (
	"net/http"
	"online-server/dto"
	"online-server/model"
	"online-server/service"
	"online-server/utils"
	"strconv"

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
		utils.ResponseWithError(c, http.StatusBadRequest, "缺少地址参数")
		return
	}

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	// 查询账户信息
	account, err := accountService.GetAccountByAddress(address)
	if err != nil {
		utils.ResponseWithError(c, http.StatusNotFound, "查询账户失败: "+err.Error())
		return
	}

	// 返回账户信息
	utils.ResponseWithData(c, "查询账户成功", account)
}

// GetUserAccounts 获取当前登录用户导入的所有账户
//
// 需要JWT认证，从Token中获取用户名，返回该用户导入的所有账户列表
//
// 路由: GET /api/accounts
func GetUserAccounts(c *gin.Context) {
	// 从中间件中获取用户名
	username, exists := c.Get("Username")
	if !exists {
		utils.ResponseWithError(c, http.StatusUnauthorized, utils.ErrorUnauthorized)
		return
	}

	usernameStr, ok := username.(string)
	if !ok {
		utils.ResponseWithError(c, http.StatusInternalServerError, "用户信息类型错误")
		return
	}

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	// 查询该用户导入的所有账户
	accounts, err := accountService.GetAccountsByImportedBy(usernameStr)
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "查询账户列表失败: "+err.Error())
		return
	}

	// 返回账户列表
	utils.ResponseWithData(c, "查询账户列表成功", accounts)
}

// GetAllAccounts 获取系统中的所有账户(仅管理员)
//
// 需要JWT认证，且用户必须具有管理员权限
//
// 路由: GET /api/accounts/all
func GetAllAccounts(c *gin.Context) {
	// 使用CheckAdminRole检查是否有管理员权限
	if !utils.CheckAdminRole(c) {
		return
	}

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	// 查询所有账户
	accounts, err := accountService.GetAccounts()
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "查询所有账户失败: "+err.Error())
		return
	}

	// 构建包含总数的响应数据
	responseData := gin.H{
		"accounts": accounts,
		"total":    len(accounts),
	}

	// 返回所有账户信息
	utils.ResponseWithData(c, "查询所有账户成功", responseData)
}

// CreateAccount 创建新账户
//
// 需要JWT认证，从请求体中获取账户信息，创建新账户并返回结果
//
// 路由: POST /api/accounts/create
func CreateAccount(c *gin.Context) {
	// 从请求中获取账户信息
	var accountInfo dto.AccountRequest
	if err := c.ShouldBindJSON(&accountInfo); err != nil {
		utils.ResponseWithError(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	modelAccount := model.Account{
		Address:     accountInfo.Address,
		CoinType:    accountInfo.CoinType,
		Description: accountInfo.Description,
		ImportedBy:  c.GetString("Username"),
	}

	// 创建新账户
	if err := accountService.CreateAccount(&modelAccount); err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "创建账户失败: "+err.Error())
		return
	}

	// 返回成功响应
	utils.ResponseWithData(c, "创建账户成功", nil)
}

// BatchImportAccounts 批量导入账户
//
// 需要JWT认证，从请求体中获取批量导入的账户信息，创建新账户并返回结果
//
// 路由: POST /api/accounts/import
func BatchImportAccounts(c *gin.Context) {
	// 从请求中获取批量导入的账户信息
	var batchImportRequest dto.BatchImportRequest
	if err := c.ShouldBindJSON(&batchImportRequest); err != nil {
		utils.ResponseWithError(c, http.StatusBadRequest, "请求参数错误: "+err.Error())
		return
	}

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	var accounts []model.Account
	for _, accountInfo := range batchImportRequest.Accounts {
		modelAccount := model.Account{
			Address:     accountInfo.Address,
			CoinType:    accountInfo.CoinType,
			Description: accountInfo.Description,
			ImportedBy:  c.GetString("Username"),
		}
		accounts = append(accounts, modelAccount)
	}

	// 批量导入账户
	if err := accountService.BatchCreateAccounts(accounts); err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "批量导入账户失败: "+err.Error())
		return
	}

	// 返回成功响应
	utils.ResponseWithData(c, "批量导入账户成功", nil)
}

// DeleteAccount 删除账户(仅管理员)
//
// 需要JWT认证，且用户必须具有管理员权限
// 通过URL参数获取账户ID，删除对应的账户
//
// 路由: DELETE /api/accounts/admin/:id
func DeleteAccount(c *gin.Context) {
	// 使用CheckAdminRole检查是否有管理员权限
	if !utils.CheckAdminRole(c) {
		return
	}

	// 获取URL中的ID参数
	idStr := c.Param("id")
	if idStr == "" {
		utils.ResponseWithError(c, http.StatusBadRequest, "缺少账户ID参数")
		return
	}

	// 转换ID为uint类型
	accountID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.ResponseWithError(c, http.StatusBadRequest, "无效的账户ID格式")
		return
	}

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	// 删除账户
	if err := accountService.DeleteAccount(uint(accountID)); err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "删除账户失败: "+err.Error())
		return
	}

	// 返回成功响应
	utils.ResponseWithData(c, "删除账户成功", nil)
}
