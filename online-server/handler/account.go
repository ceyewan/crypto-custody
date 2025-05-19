package handler

import (
	"net/http"
	"strconv"
	"time"

	"online-server/dto"
	"online-server/model"
	"online-server/service"

	"github.com/gin-gonic/gin"
)

// GetAllAccounts 获取所有账户列表，仅管理员可访问
func GetAllAccounts(c *gin.Context) {
	// 获取当前用户
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未找到用户信息"})
		return
	}

	// 检查用户角色
	currentUser := user.(*model.User)
	if currentUser.Role != model.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，需要管理员权限"})
		return
	}

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "初始化账户服务失败: " + err.Error()})
		return
	}

	// 获取所有账户
	accounts, err := accountService.GetAccounts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取账户列表失败: " + err.Error()})
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

	c.JSON(http.StatusOK, response)
}

// GetUserAccounts 获取当前用户的账户列表
func GetUserAccounts(c *gin.Context) {
	// 获取当前用户
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未找到用户信息"})
		return
	}

	currentUser := user.(*model.User)

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "初始化账户服务失败: " + err.Error()})
		return
	}

	var accounts []model.Account
	// 管理员可以看所有账户，警员只能看自己的账户
	if currentUser.Role == model.RoleAdmin {
		accounts, err = accountService.GetAccounts()
	} else {
		accounts, err = accountService.GetAccountsByUserID(currentUser.ID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取账户列表失败: " + err.Error()})
		return
	}

	// 构建响应
	var response []dto.AccountResponse
	for _, account := range accounts {
		response = append(response, dto.AccountResponse{
			ID:          account.ID,
			Address:     account.Address,
			CoinType:    account.CoinType,
			Balance:     account.Balance,
			ImportedBy:  account.ImportedBy,
			Description: account.Description,
		})
	}

	c.JSON(http.StatusOK, response)
}

// GetAccountByID 通过ID获取账户详情
func GetAccountByID(c *gin.Context) {
	// 获取当前用户
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未找到用户信息"})
		return
	}

	currentUser := user.(*model.User)

	// 获取账户ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的账户ID"})
		return
	}

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "初始化账户服务失败: " + err.Error()})
		return
	}

	// 获取账户信息
	account, err := accountService.GetAccountByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定的账户: " + err.Error()})
		return
	}

	// 验证权限：非管理员只能查看自己的账户
	if currentUser.Role != model.RoleAdmin && account.ImportedBy != currentUser.Username {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，您只能查看自己的账户"})
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

	c.JSON(http.StatusOK, response)
}

// CreateAccount 创建新账户
func CreateAccount(c *gin.Context) {
	// 获取当前用户
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未找到用户信息"})
		return
	}

	currentUser := user.(*model.User)

	// 只有管理员可以创建账户
	if currentUser.Role != model.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，需要管理员权限"})
		return
	}

	// 解析请求体
	var req dto.AccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "初始化账户服务失败: " + err.Error()})
		return
	}

	// 创建账户
	account := model.Account{
		Address:     req.Address,
		CoinType:    req.CoinType,
		Balance:     req.Balance,
		ImportedBy:  currentUser.Username,
		UserID:      req.UserID,
		Description: req.Description,
	}

	if err := accountService.CreateAccount(&account); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建账户失败: " + err.Error()})
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

	c.JSON(http.StatusCreated, response)
}

// UpdateAccount 更新账户信息
func UpdateAccount(c *gin.Context) {
	// 获取当前用户
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未找到用户信息"})
		return
	}

	currentUser := user.(*model.User)

	// 只有管理员可以更新账户
	if currentUser.Role != model.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，需要管理员权限"})
		return
	}

	// 获取账户ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的账户ID"})
		return
	}

	// 解析请求体
	var req dto.AccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "初始化账户服务失败: " + err.Error()})
		return
	}

	// 获取现有账户
	account, err := accountService.GetAccountByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定的账户: " + err.Error()})
		return
	}

	// 更新账户信息
	account.Address = req.Address
	account.CoinType = req.CoinType
	account.Balance = req.Balance
	account.UserID = req.UserID
	account.Description = req.Description

	if err := accountService.UpdateAccount(account); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新账户失败: " + err.Error()})
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

	c.JSON(http.StatusOK, response)
}

// UpdateAccountBalance 更新账户余额
func UpdateAccountBalance(c *gin.Context) {
	// 获取当前用户
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未找到用户信息"})
		return
	}

	currentUser := user.(*model.User)

	// 只有管理员可以更新账户余额
	if currentUser.Role != model.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，需要管理员权限"})
		return
	}

	// 获取账户ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的账户ID"})
		return
	}

	// 解析请求体
	var req dto.AccountUpdateBalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "初始化账户服务失败: " + err.Error()})
		return
	}

	// 更新账户余额
	if err := accountService.UpdateAccountBalance(uint(id), req.Balance); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新账户余额失败: " + err.Error()})
		return
	}

	// 获取更新后的账户信息
	account, err := accountService.GetAccountByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "获取更新后的账户信息失败: " + err.Error()})
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

	c.JSON(http.StatusOK, response)
}

// DeleteAccount 删除账户
func DeleteAccount(c *gin.Context) {
	// 获取当前用户
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未找到用户信息"})
		return
	}

	currentUser := user.(*model.User)

	// 只有管理员可以删除账户
	if currentUser.Role != model.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，需要管理员权限"})
		return
	}

	// 获取账户ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的账户ID"})
		return
	}

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "初始化账户服务失败: " + err.Error()})
		return
	}

	// 删除账户
	if err := accountService.DeleteAccount(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除账户失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "账户删除成功"})
}

// BatchImportAccounts 批量导入账户
func BatchImportAccounts(c *gin.Context) {
	// 获取当前用户
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未找到用户信息"})
		return
	}

	currentUser := user.(*model.User)

	// 只有管理员可以批量导入账户
	if currentUser.Role != model.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足，需要管理员权限"})
		return
	}

	// 解析请求体
	var req dto.BatchImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据: " + err.Error()})
		return
	}

	// 检查是否有账户数据
	if len(req.Accounts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有提供账户数据"})
		return
	}

	// 获取账户服务实例
	accountService, err := service.GetAccountServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "初始化账户服务失败: " + err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "批量导入账户失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "批量导入账户成功", "count": len(accounts)})
}
