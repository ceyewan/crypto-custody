package handler

import (
	"errors"
	"net/http"
	"online-server/dto"
	"online-server/model"
	"online-server/service"
	"online-server/utils"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Login 处理用户登录请求
//
// 通过用户名和密码验证用户身份，并颁发JWT令牌
//
// 参数：
// - c：Gin上下文，包含HTTP请求和响应
//
// 响应：
// - 200 OK：登录成功，返回用户信息和JWT令牌
// - 400 Bad Request：请求参数不正确
// - 401 Unauthorized：用户名或密码错误
// - 500 Internal Server Error：服务器内部错误
func Login(c *gin.Context) {
	var loginReq dto.LoginRequest

	// 绑定并验证请求数据
	if !utils.BindJSON(c, &loginReq) {
		return
	}

	// 调用用户服务处理登录
	userService, err := service.GetUserServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	user, token, err := userService.Login(loginReq.Username, loginReq.Password)
	if err != nil {
		utils.ResponseWithError(c, http.StatusUnauthorized, err.Error())
		return
	}

	// 准备响应数据
	userResp := dto.UserResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     string(user.Role),
	}

	utils.ResponseWithData(c, "登录成功", dto.LoginResponse{
		Token: token,
		User:  userResp,
	})
}

// Register 处理用户注册请求
//
// 创建新用户账户，验证用户名和邮箱的唯一性
//
// 参数：
// - c：Gin上下文，包含HTTP请求和响应
//
// 响应：
// - 200 OK：注册成功，返回新用户信息
// - 400 Bad Request：请求参数不正确或用户名/邮箱已存在
// - 500 Internal Server Error：服务器内部错误
func Register(c *gin.Context) {
	var registerReq dto.RegisterRequest

	// 绑定并验证请求数据
	if !utils.BindJSON(c, &registerReq) {
		return
	}

	// 调用用户服务处理注册
	userService, err := service.GetUserServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	user, err := userService.Register(registerReq.Username, registerReq.Password, registerReq.Email)
	if err != nil {
		utils.ResponseWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	// 准备响应数据
	userResp := dto.UserResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     string(user.Role),
	}

	utils.ResponseWithData(c, "注册成功", userResp)
}

// Logout 处理用户登出请求
//
// 将当前用户的JWT令牌加入黑名单，使其失效
//
// 参数：
// - c：Gin上下文，包含HTTP请求和响应
//
// 响应：
// - 200 OK：登出成功
// - 500 Internal Server Error：服务器内部错误
func Logout(c *gin.Context) {
	token := c.GetHeader("Authorization")

	userService, err := service.GetUserServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	userService.Logout(token)

	utils.ResponseWithSuccess(c, "登出成功")
}

// GetUsers 获取系统中的所有用户
//
// 仅管理员用户可调用此端点，返回系统中所有用户的列表
//
// 参数：
// - c：Gin上下文，包含HTTP请求和响应
//
// 响应：
// - 200 OK：成功获取用户列表
// - 403 Forbidden：当前用户无管理员权限
// - 500 Internal Server Error：服务器内部错误
func GetUsers(c *gin.Context) {
	// 检查管理员权限
	if !utils.CheckAdminRole(c) {
		return
	}

	userService, err := service.GetUserServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	users, err := userService.GetUsers()
	if utils.HandleServiceError(c, err, "获取用户列表失败") {
		return
	}

	// 转换为响应格式
	var userList []dto.UserResponse
	for _, user := range users {
		userList = append(userList, dto.UserResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     string(user.Role),
		})
	}

	utils.ResponseWithData(c, "获取用户列表成功", userList)
}

// GetUserByID 根据ID获取用户详细信息
//
// 仅管理员可调用此端点，通过用户ID获取特定用户的详细信息
//
// 参数：
// - c：Gin上下文，包含HTTP请求和响应，其中包括路径参数"id"
//
// 响应：
// - 200 OK：成功获取用户信息
// - 400 Bad Request：提供的用户ID格式不正确
// - 404 Not Found：指定ID的用户不存在
// - 500 Internal Server Error：服务器内部错误
func GetUserByID(c *gin.Context) {
	// 获取用户ID参数
	userID, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}

	userService, err := service.GetUserServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	user, err := userService.GetUserByID(userID)
	if utils.HandleServiceError(c, err, "用户不存在") {
		return
	}

	// 准备响应数据
	userResp := dto.UserResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     string(user.Role),
	}

	utils.ResponseWithData(c, "获取用户信息成功", userResp)
}

// ChangePassword 处理用户修改自身密码的请求
//
// 验证当前密码并设置新密码
//
// 参数：
// - c：Gin上下文，包含HTTP请求和响应
//
// 响应：
// - 200 OK：密码修改成功
// - 400 Bad Request：请求参数不正确或当前密码验证失败
// - 401 Unauthorized：用户未认证
// - 500 Internal Server Error：服务器内部错误
func ChangePassword(c *gin.Context) {
	var changePasswordReq dto.ChangePasswordRequest

	if !utils.BindJSON(c, &changePasswordReq) {
		return
	}

	// 获取当前用户名
	userName := c.GetString("Username")

	userService, err := service.GetUserServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	// 获取当前用户ID
	user, err := userService.GetUserByUsername(userName)
	if err != nil {
		utils.ResponseWithError(c, http.StatusUnauthorized, utils.ErrorUserNotFound)
		return
	}

	if err := userService.ChangePassword(user.ID, changePasswordReq.OldPassword, changePasswordReq.NewPassword); err != nil {
		utils.ResponseWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.ResponseWithSuccess(c, "密码修改成功")
}

// GetCurrentUser 获取当前登录用户的详细信息
//
// 通过JWT令牌中的用户名获取当前已认证用户的详细信息
//
// 参数：
// - c：Gin上下文，包含HTTP请求和响应
//
// 响应：
// - 200 OK：成功获取用户信息
// - 500 Internal Server Error：服务器内部错误
func GetCurrentUser(c *gin.Context) {
	// 从中间件中获取用户信息
	userObj, exists := c.Get("user")
	if !exists {
		utils.ResponseWithError(c, http.StatusUnauthorized, utils.ErrorUserNotFound)
		return
	}

	user, ok := userObj.(*model.User)
	if !ok {
		utils.ResponseWithError(c, http.StatusInternalServerError, "用户信息类型错误")
		return
	}

	// 准备响应数据
	userResp := dto.UserResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     string(user.Role),
	}

	utils.ResponseWithData(c, "获取当前用户信息成功", userResp)
}

// UpdateUserRole 更新用户角色
//
// 仅管理员可以修改其他非管理员用户的角色
//
// 参数：
// - c：Gin上下文，包含HTTP请求和响应，其中包括路径参数"id"
//
// 响应：
// - 200 OK：角色更新成功
// - 400 Bad Request：请求参数不正确或角色值无效
// - 403 Forbidden：当前用户无管理员权限或尝试修改管理员角色
// - 500 Internal Server Error：服务器内部错误
func UpdateUserRole(c *gin.Context) {
	// 检查管理员权限
	if !utils.CheckAdminRole(c) {
		return
	}

	// 获取用户ID参数
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseWithError(c, http.StatusBadRequest, utils.ErrorInvalidID)
		return
	}

	var roleReq dto.UpdateRoleRequest

	if !utils.BindJSON(c, &roleReq) {
		return
	}

	// 验证角色值是否有效
	var newRole model.Role
	switch model.Role(roleReq.Role) {
	case model.RoleAdmin, model.RoleOfficer, model.RoleGuest:
		newRole = model.Role(roleReq.Role)
	default:
		utils.ResponseWithError(c, http.StatusBadRequest, "无效的角色值")
		return
	}

	userService, err := service.GetUserServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	err = userService.UpdateUserRole(uint(userID), newRole)
	if err != nil {
		utils.ResponseWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.ResponseWithSuccess(c, "用户角色更新成功")
}

// DeleteUser 删除用户
//
// 仅管理员可以删除用户账户
//
// 参数：
// - c：Gin上下文，包含HTTP请求和响应，其中包括路径参数"id"
//
// 响应：
// - 200 OK：用户删除成功
// - 400 Bad Request：无效的用户ID
// - 403 Forbidden：当前用户无管理员权限
// - 500 Internal Server Error：服务器内部错误
func DeleteUser(c *gin.Context) {
	// 检查管理员权限
	if !utils.CheckAdminRole(c) {
		return
	}

	// 获取用户ID参数
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseWithError(c, http.StatusBadRequest, utils.ErrorInvalidID)
		return
	}

	userService, err := service.GetUserServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	err = userService.DeleteUser(uint(userID))
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "删除用户失败: "+err.Error())
		return
	}

	utils.ResponseWithSuccess(c, "用户删除成功")
}

// CheckAuth 验证Token是否有效
//
// 检查提供的JWT令牌是否有效，并返回令牌中包含的用户信息
//
// 参数：
// - c：Gin上下文，包含HTTP请求和响应
//
// 响应：
// - 200 OK：令牌有效，返回用户信息
// - 400 Bad Request：请求参数不正确
// - 401 Unauthorized：令牌无效或用户不存在
// - 500 Internal Server Error：服务器内部错误
func CheckAuth(c *gin.Context) {
	var authReq dto.CheckAuthRequest

	if !utils.BindJSON(c, &authReq) {
		// 这个特殊情况，我们需要构造一个 AuthResponse
		c.JSON(http.StatusBadRequest, dto.AuthResponse{
			StandardResponse: dto.StandardResponse{
				Code:    http.StatusBadRequest,
				Message: utils.ErrorBadRequest,
			},
			Valid: false,
		})
		return
	}

	valid, username, role := utils.CheckAuth(authReq.Token)

	if !valid {
		c.JSON(http.StatusUnauthorized, dto.AuthResponse{
			StandardResponse: dto.StandardResponse{
				Code:    http.StatusUnauthorized,
				Message: "令牌无效",
			},
			Valid: false,
		})
		return
	}

	userService, err := service.GetUserServiceInstance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.AuthResponse{
			StandardResponse: dto.StandardResponse{
				Code:    http.StatusInternalServerError,
				Message: utils.ErrorInternalServerError + ": " + err.Error(),
			},
			Valid: false,
		})
		return
	}

	user, err := userService.GetUserByUsername(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.AuthResponse{
			StandardResponse: dto.StandardResponse{
				Code:    http.StatusUnauthorized,
				Message: utils.ErrorUserNotFound,
			},
			Valid: false,
		})
		return
	}

	// 准备响应数据
	userResp := dto.UserResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     role,
	}

	// 注意：这里仍然使用 c.JSON，因为需要返回自定义的 AuthResponse 类型
	c.JSON(http.StatusOK, dto.AuthResponse{
		StandardResponse: dto.StandardResponse{
			Code:    http.StatusOK,
			Message: "令牌有效",
			Data:    map[string]interface{}{"user": userResp},
		},
		Valid: true,
	})
}

// UpdateUserID 管理员更新用户名
//
// 仅管理员可以修改其他非管理员用户的用户名
//
// 参数：
// - c：Gin上下文，包含HTTP请求和响应，其中包括路径参数"id"
//
// 响应：
// - 200 OK：用户名更新成功
// - 400 Bad Request：请求参数不正确、无效的用户ID或用户名已存在
// - 403 Forbidden：当前用户无管理员权限或尝试修改管理员用户名
// - 500 Internal Server Error：服务器内部错误
func UpdateUserID(c *gin.Context) {
	// 检查管理员权限
	if !utils.CheckAdminRole(c) {
		return
	}

	// 获取用户ID参数
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseWithError(c, http.StatusBadRequest, utils.ErrorInvalidID)
		return
	}

	var usernameReq dto.UpdateUsernameRequest

	if !utils.BindJSON(c, &usernameReq) {
		return
	}

	userService, err := service.GetUserServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	err = userService.UpdateUserID(uint(userID), usernameReq.Username)
	if err != nil {
		utils.ResponseWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.ResponseWithSuccess(c, "用户名更新成功")
}

// AdminChangePassword 管理员修改用户密码
//
// 管理员可以直接修改任何用户的密码，无需验证当前密码
//
// 参数：
// - c：Gin上下文，包含HTTP请求和响应，其中包括路径参数"id"
//
// 响应：
// - 200 OK：密码修改成功
// - 400 Bad Request：请求参数不正确
// - 403 Forbidden：当前用户无管理员权限
// - 404 Not Found：指定ID的用户不存在
// - 500 Internal Server Error：服务器内部错误
func AdminChangePassword(c *gin.Context) {
	// 检查管理员权限
	if !utils.CheckAdminRole(c) {
		return
	}

	// 获取用户ID参数
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		utils.ResponseWithError(c, http.StatusBadRequest, utils.ErrorInvalidID)
		return
	}

	var passwordReq dto.AdminChangePasswordRequest
	if !utils.BindJSON(c, &passwordReq) {
		return
	}

	userService, err := service.GetUserServiceInstance()
	if utils.HandleServiceInitError(c, err) {
		return
	}

	err = userService.AdminChangePassword(uint(userID), passwordReq.NewPassword)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.ResponseWithError(c, http.StatusNotFound, utils.ErrorRecordNotFound)
			return
		}
		utils.ResponseWithError(c, http.StatusBadRequest, err.Error())
		return
	}

	utils.ResponseWithSuccess(c, "用户密码修改成功")
}
