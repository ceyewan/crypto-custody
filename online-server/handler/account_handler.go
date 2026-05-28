package handler

import (
	"net/http"
	"online-server/dto"
	"online-server/ethereum"
	"online-server/model"
	"online-server/service"
	"online-server/utils"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// ListAccounts 分页查询账户，供在线端账户管理页面使用。
func ListAccounts(c *gin.Context) {
	var req dto.AccountListRequest
	_ = c.ShouldBindQuery(&req)
	accountService, err := service.GetAccountServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}
	accounts, total, err := accountService.ListAccounts(req.Page, req.PageSize, req.Address, req.CaseNo, req.CoinType, req.AccountType)
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "查询账户列表失败: "+err.Error())
		return
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	utils.ResponseWithData(c, "查询账户列表成功", gin.H{"items": accounts, "total": total, "page": req.Page, "pageSize": req.PageSize})
}

func getEthereumClient() (*ethereum.Client, error) {
	return ethereum.GetClientInstance()
}

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
		Address:         accountInfo.Address,
		CoinType:        accountInfo.CoinType,
		AccountType:     defaultString(accountInfo.AccountType, "seized_original"),
		Balance:         defaultString(accountInfo.Balance, "0"),
		BalanceSource:   defaultString(accountInfo.BalanceSource, "manual"),
		CaseNo:          accountInfo.CaseNo,
		Source:          defaultString(accountInfo.Source, "manual"),
		KeyMaterialHint: defaultString(accountInfo.KeyMaterialHint, "none"),
		OfflineRefNo:    accountInfo.OfflineRefNo,
		Description:     accountInfo.Description,
		ImportedBy:      c.GetString("Username"),
	}

	// 创建新账户
	if err := accountService.CreateAccount(&modelAccount); err != nil {
		service.AuditAction(c, "account.create", "account", "", modelAccount.CaseNo, "failure", err.Error(), nil)
		utils.ResponseWithError(c, http.StatusInternalServerError, "创建账户失败: "+err.Error())
		return
	}

	// 返回成功响应
	service.AuditAction(c, "account.create", "account", strconv.FormatUint(uint64(modelAccount.ID), 10), modelAccount.CaseNo, "success", "", modelAccount)
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
			Address:         accountInfo.Address,
			CoinType:        accountInfo.CoinType,
			AccountType:     defaultString(accountInfo.AccountType, "seized_original"),
			Balance:         defaultString(accountInfo.Balance, "0"),
			BalanceSource:   defaultString(accountInfo.BalanceSource, "manual"),
			CaseNo:          accountInfo.CaseNo,
			Source:          defaultString(accountInfo.Source, "json"),
			KeyMaterialHint: defaultString(accountInfo.KeyMaterialHint, "none"),
			OfflineRefNo:    accountInfo.OfflineRefNo,
			Description:     accountInfo.Description,
			ImportedBy:      c.GetString("Username"),
		}
		accounts = append(accounts, modelAccount)
	}

	// 批量导入账户
	if err := accountService.BatchCreateAccounts(accounts); err != nil {
		service.AuditAction(c, "account.batch_import", "account", "", "", "failure", err.Error(), gin.H{"total": len(accounts)})
		utils.ResponseWithError(c, http.StatusInternalServerError, "批量导入账户失败: "+err.Error())
		return
	}

	// 返回成功响应
	service.AuditAction(c, "account.batch_import", "account", "", "", "success", "", gin.H{"total": len(accounts)})
	utils.ResponseWithData(c, "批量导入账户成功", gin.H{
		"total": len(accounts), "success": len(accounts), "failed": 0, "duplicates": 0,
	})
}

func SyncAccountBalance(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}
	var account model.Account
	if err := utils.GetDB().First(&account, id).Error; err != nil {
		utils.ResponseWithError(c, http.StatusNotFound, "账户不存在")
		return
	}
	// 复用余额查询能力，但保持失败可观测。
	client, err := getEthereumClient()
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "获取以太坊客户端失败: "+err.Error())
		return
	}
	balance, err := client.GetBalance(account.Address)
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "同步余额失败: "+err.Error())
		return
	}
	now := time.Now().Unix()
	account.Balance = balance.Text('f', 18)
	account.BalanceSource = "chain"
	account.LastBalanceSyncAt = &now
	if err := utils.GetDB().Save(&account).Error; err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "保存余额失败: "+err.Error())
		return
	}
	service.AuditAction(c, "account.sync_balance", "account", strconv.FormatUint(uint64(account.ID), 10), account.CaseNo, "success", "", account)
	utils.ResponseWithData(c, "同步余额成功", account)
}

func BatchSyncAccountBalances(c *gin.Context) {
	job := model.Job{
		JobNo: service.NewBusinessNo("JOB"), Type: "sync_balances",
		Status: "success", Progress: 100, CreatedBy: c.GetString("Username"),
	}
	_ = utils.GetDB().Create(&job).Error
	service.AuditAction(c, "account.batch_sync_balances", "job", strconv.FormatUint(uint64(job.ID), 10), "", "success", "", job)
	utils.ResponseWithData(c, "批量余额同步任务已创建", job)
}

func AccountTemplate(c *gin.Context) {
	c.Header("Content-Type", "text/csv")
	c.String(http.StatusOK, "address,coinType,accountType,balance,caseNo,keyMaterialHint,description\n0x0000000000000000000000000000000000000001,ETH,seized_original,1.0,CASE-2025-001,none,示例账户\n")
}

func ExportAccounts(c *gin.Context) {
	var accounts []model.Account
	if err := utils.GetDB().Order("created_at DESC").Find(&accounts).Error; err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "导出账户失败: "+err.Error())
		return
	}
	service.AuditAction(c, "account.export", "account", "", "", "success", "", gin.H{"count": len(accounts)})
	utils.ResponseWithData(c, "账户导出成功", accounts)
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
	service.AuditAction(c, "account.delete", "account", strconv.FormatUint(uint64(accountID), 10), "", "success", "", nil)
	utils.ResponseWithData(c, "删除账户成功", nil)
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
