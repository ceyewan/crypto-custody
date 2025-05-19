package handler

import (
	"net/http"
	"time"

	"online-server/dto"
	"online-server/model"
	"online-server/service"
	"online-server/utils"

	"github.com/gin-gonic/gin"
)

// GetAccountByAddress 通过地址获取账户信息
func GetAccountByAddress(c *gin.Context) {
	// 获取地址参数
	address := c.Param("address")
	if address == "" {
		utils.ResponseWithError(c, http.StatusBadRequest, "未提供账户地址")
		return
	}

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	// 获取账户信息
	account, err := accountService.GetAccountByAddress(address)
	if utils.HandleServiceError(c, err, "未找到指定地址的账户") {
		return
	}

	// 构建响应
	response := dto.AccountResponse{
		Address:     account.Address,
		CoinType:    account.CoinType,
		Balance:     account.Balance,
		ImportedBy:  account.ImportedBy,
		Description: account.Description,
	}

	utils.ResponseWithData(c, "获取账户详情成功", response)
}

// GetAllAccounts 获取所有账户列表，仅管理员可访问
func GetAllAccounts(c *gin.Context) {
	// 检查管理员权限
	if !utils.CheckAdminRole(c) {
		return
	}

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	// 获取所有账户
	accounts, err := accountService.GetAccounts()
	if utils.HandleServiceError(c, err, "获取账户列表失败") {
		return
	}

	// 构建响应
	var response []dto.AccountResponse
	for _, account := range accounts {
		response = append(response, dto.AccountResponse{
			Address:     account.Address,
			CoinType:    account.CoinType,
			Balance:     account.Balance,
			ImportedBy:  account.ImportedBy,
			Description: account.Description,
		})
	}

	utils.ResponseWithData(c, "获取账户列表成功", response)
}

// GetUserAccounts 获取当前用户的账户列表
func GetUserAccounts(c *gin.Context) {
	// 获取当前用户
	currentUser, ok := utils.GetCurrentUser(c)
	if !ok {
		return
	}

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	// 只获取用户自己的账户
	accounts, err := accountService.GetAccountsByImportedBy(currentUser.Username)
	if utils.HandleServiceError(c, err, "获取账户列表失败") {
		return
	}

	// 构建响应
	var response []dto.AccountResponse
	for _, account := range accounts {
		response = append(response, dto.AccountResponse{
			Address:     account.Address,
			CoinType:    account.CoinType,
			Balance:     account.Balance,
			ImportedBy:  account.ImportedBy,
			Description: account.Description,
		})
	}

	utils.ResponseWithData(c, "获取账户列表成功", response)
}

// GetAccountByID 通过ID获取账户详情
func GetAccountByID(c *gin.Context) {
	// 获取当前用户
	currentUser, ok := utils.GetCurrentUser(c)
	if !ok {
		return
	}

	// 获取账户ID
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	// 获取账户信息
	account, err := accountService.GetAccountByID(id)
	if utils.HandleServiceError(c, err, "未找到指定的账户") {
		return
	}

	// 验证权限：非管理员只能查看自己的账户
	if currentUser.Role != model.RoleAdmin && account.ImportedBy != currentUser.Username {
		utils.ResponseWithError(c, http.StatusForbidden, "权限不足，您只能查看自己的账户")
		return
	}

	// 构建响应
	response := dto.AccountResponse{
		Address:     account.Address,
		CoinType:    account.CoinType,
		Balance:     account.Balance,
		ImportedBy:  account.ImportedBy,
		Description: account.Description,
	}

	utils.ResponseWithData(c, "获取账户详情成功", response)
}

// CreateAccount 创建新账户
func CreateAccount(c *gin.Context) {
	// 检查管理员权限
	if !utils.CheckAdminRole(c) {
		return
	}

	currentUser, _ := utils.GetCurrentUser(c)

	// 解析请求体
	var req dto.AccountRequest
	if !utils.BindJSON(c, &req) {
		return
	}

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	// 创建账户
	account := model.Account{
		Address:     req.Address,
		CoinType:    req.CoinType,
		Balance:     "0",
		ImportedBy:  currentUser.Username,
		Description: req.Description,
	}

	if err := accountService.CreateAccount(&account); err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "创建账户失败: "+err.Error())
		return
	}

	// 构建响应
	response := dto.AccountResponse{
		Address:     account.Address,
		CoinType:    account.CoinType,
		Balance:     account.Balance,
		ImportedBy:  account.ImportedBy,
		UserID:      account.UserID,
		Description: account.Description,
		CreatedAt:   account.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   account.UpdatedAt.Format(time.RFC3339),
	}

	c.JSON(http.StatusCreated, dto.StandardResponse{
		Code:    http.StatusCreated,
		Message: "创建账户成功",
		Data:    response,
	})
}

// UpdateAccount 更新账户信息
func UpdateAccount(c *gin.Context) {
	// 检查管理员权限
	if !utils.CheckAdminRole(c) {
		return
	}

	// 获取账户ID
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}

	// 解析请求体
	var req dto.AccountRequest
	if !utils.BindJSON(c, &req) {
		return
	}

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	// 获取现有账户
	account, err := accountService.GetAccountByID(id)
	if utils.HandleServiceError(c, err, "未找到指定的账户") {
		return
	}

	// 更新账户信息
	account.Address = req.Address
	account.CoinType = req.CoinType
	account.Balance = req.Balance
	account.UserID = req.UserID
	account.Description = req.Description

	if err := accountService.UpdateAccount(account); err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "更新账户失败: "+err.Error())
		return
	}

	// 构建响应
	response := dto.AccountResponse{
		ID:          account.ID,
		Address:     account.Address,
		CoinType:    account.CoinType,
		Balance:     account.Balance,
		ImportedBy:  account.ImportedBy,
		UserID:      account.UserID,
		Description: account.Description,
		CreatedAt:   account.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   account.UpdatedAt.Format(time.RFC3339),
	}

	utils.ResponseWithData(c, "更新账户成功", response)
}

// UpdateAccountBalance 更新账户余额
func UpdateAccountBalance(c *gin.Context) {
	// 检查管理员权限
	if !utils.CheckAdminRole(c) {
		return
	}

	// 获取账户ID
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}

	// 解析请求体
	var req dto.AccountUpdateBalanceRequest
	if !utils.BindJSON(c, &req) {
		return
	}

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	// 更新账户余额
	if err := accountService.UpdateAccountBalance(id, req.Balance); err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "更新账户余额失败: "+err.Error())
		return
	}

	// 获取更新后的账户信息
	account, err := accountService.GetAccountByID(id)
	if utils.HandleServiceError(c, err, "获取更新后的账户信息失败") {
		return
	}

	// 构建响应
	response := dto.AccountResponse{
		ID:          account.ID,
		Address:     account.Address,
		CoinType:    account.CoinType,
		Balance:     account.Balance,
		ImportedBy:  account.ImportedBy,
		UserID:      account.UserID,
		Description: account.Description,
		CreatedAt:   account.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   account.UpdatedAt.Format(time.RFC3339),
	}

	utils.ResponseWithData(c, "更新账户余额成功", response)
}

// DeleteAccount 删除账户
func DeleteAccount(c *gin.Context) {
	// 检查管理员权限
	if !utils.CheckAdminRole(c) {
		return
	}

	// 获取账户ID
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	// 删除账户
	if err := accountService.DeleteAccount(id); err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "删除账户失败: "+err.Error())
		return
	}

	utils.ResponseWithSuccess(c, "账户删除成功")
}

// BatchImportAccounts 批量导入账户
func BatchImportAccounts(c *gin.Context) {
	// 检查管理员权限
	if !utils.CheckAdminRole(c) {
		return
	}

	currentUser, _ := utils.GetCurrentUser(c)

	// 解析请求体
	var req dto.BatchImportRequest
	if !utils.BindJSON(c, &req) {
		return
	}

	// 检查是否有账户数据
	if len(req.Accounts) == 0 {
		utils.ResponseWithError(c, http.StatusBadRequest, "没有提供账户数据")
		return
	}

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	// 转换请求数据为账户模型
	var accounts []model.Account
	for _, reqAccount := range req.Accounts {
		account := model.Account{
			Address:     reqAccount.Address,
			CoinType:    reqAccount.CoinType,
			Balance:     reqAccount.Balance,
			ImportedBy:  currentUser.Username,
			UserID:      reqAccount.UserID,
			Description: reqAccount.Description,
		}
		accounts = append(accounts, account)
	}

	// 批量导入账户
	if err := accountService.BatchCreateAccounts(accounts); err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "批量导入账户失败: "+err.Error())
		return
	}

	utils.ResponseWithData(c, "批量导入账户成功", gin.H{"count": len(accounts)})
}
