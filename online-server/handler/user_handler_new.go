package handler

import (
	"errors"
	"fmt"
	"net/http"
	"online-server/dto"
	"online-server/model"
	"online-server/service"
	"online-server/utils"
	"strconv"

	"gorm.io/gorm"
)

// UserHandler 用户处理器
type UserHandler struct {
	*Handler
}

// NewUserHandler 创建用户处理器
func NewUserHandler() *UserHandler {
	return &UserHandler{
		Handler: NewHandler("user"),
	}
}

// Login 处理用户登录请求
func (h *UserHandler) Login() gin.HandlerFunc {
	return h.Handle(func(c *Context) error {
		var req dto.LoginRequest
		if !c.BindRequest(&req) {
			return nil
		}

		userService, err := service.GetUserServiceInstance()
		if err != nil {
			c.ServerError("系统错误", err.Error())
			return err
		}

		user, token, err := userService.Login(req.Username, req.Password)
		if err != nil {
			c.Unauthorized(err.Error())
			return err
		}

		userResp := dto.UserResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     string(user.Role),
		}

		c.Data("登录成功", dto.LoginResponse{
			Token: token,
			User:  userResp,
		})
		return nil
	})
}

// Register 处理用户注册请求
func (h *UserHandler) Register() gin.HandlerFunc {
	return h.Handle(func(c *Context) error {
		var req dto.RegisterRequest
		if !c.BindRequest(&req) {
			return nil
		}

		userService, err := service.GetUserServiceInstance()
		if err != nil {
			c.ServerError("系统错误", err.Error())
			return err
		}

		user, err := userService.Register(req.Username, req.Password, req.Email)
		if err != nil {
			c.BadRequest(err.Error())
			return err
		}

		userResp := dto.UserResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     string(user.Role),
		}

		c.Data("注册成功", userResp)
		return nil
	})
}

// Logout 处理用户登出请求
func (h *UserHandler) Logout() gin.HandlerFunc {
	return h.Handle(func(c *Context) error {
		token := c.GetHeader("Authorization")

		userService, err := service.GetUserServiceInstance()
		if err != nil {
			c.ServerError("系统错误", err.Error())
			return err
		}

		userService.Logout(token)
		c.Success("登出成功")
		return nil
	})
}

// GetUsers 获取系统中的所有用户
func (h *UserHandler) GetUsers() gin.HandlerFunc {
	return h.Handle(func(c *Context) error {
		// 检查管理员权限
		if !c.RequireAdmin() {
			return fmt.Errorf("需要管理员权限")
		}

		// 获取分页参数
		page, pageSize := c.GetPageRequest()

		userService, err := service.GetUserServiceInstance()
		if err != nil {
			c.ServerError("系统错误", err.Error())
			return err
		}

		users, total, err := userService.GetUsersPaginated(page, pageSize)
		if err != nil {
			c.ServerError("获取用户列表失败", err.Error())
			return err
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

		c.Pagination("获取用户列表成功", userList, total, page, pageSize)
		return nil
	})
}

// GetUserByID 根据ID获取用户详细信息
func (h *UserHandler) GetUserByID() gin.HandlerFunc {
	return h.Handle(func(c *Context) error {
		// 检查管理员权限
		if !c.RequireAdmin() {
			return fmt.Errorf("需要管理员权限")
		}

		// 获取用户ID参数
		userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
		if err != nil {
			c.BadRequest("无效的用户ID", err.Error())
			return err
		}

		userService, err := service.GetUserServiceInstance()
		if err != nil {
			c.ServerError("系统错误", err.Error())
			return err
		}

		user, err := userService.GetUserByID(uint(userID))
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.NotFound("用户不存在")
			} else {
				c.ServerError("获取用户失败", err.Error())
			}
			return err
		}

		// 准备响应数据
		userResp := dto.UserResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     string(user.Role),
		}

		c.Data("获取用户信息成功", userResp)
		return nil
	})
}

// ChangePassword 处理用户修改自身密码的请求
func (h *UserHandler) ChangePassword() gin.HandlerFunc {
	return h.Handle(func(c *Context) error {
		var req dto.ChangePasswordRequest
		if !c.BindRequest(&req) {
			return nil
		}

		userService, err := service.GetUserServiceInstance()
		if err != nil {
			c.ServerError("系统错误", err.Error())
			return err
		}

		if err := userService.ChangePassword(c.UserID, req.OldPassword, req.NewPassword); err != nil {
			c.BadRequest(err.Error())
			return err
		}

		c.Success("密码修改成功")
		return nil
	})
}

// GetCurrentUser 获取当前登录用户的详细信息
func (h *UserHandler) GetCurrentUser() gin.HandlerFunc {
	return h.Handle(func(c *Context) error {
		userService, err := service.GetUserServiceInstance()
		if err != nil {
			c.ServerError("系统错误", err.Error())
			return err
		}

		user, err := userService.GetUserByUsername(c.UserName)
		if err != nil {
			c.ServerError("获取用户信息失败", err.Error())
			return err
		}

		// 准备响应数据
		userResp := dto.UserResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     string(user.Role),
		}

		c.Data("获取当前用户信息成功", userResp)
		return nil
	})
}

// UpdateUserRole 更新用户角色
func (h *UserHandler) UpdateUserRole() gin.HandlerFunc {
	return h.Handle(func(c *Context) error {
		// 检查管理员权限
		if !c.RequireAdmin() {
			return fmt.Errorf("需要管理员权限")
		}

		// 获取用户ID参数
		userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.BadRequest("无效的用户ID", err.Error())
			return err
		}

		var req dto.UpdateRoleRequest
		if !c.BindRequest(&req) {
			return nil
		}

		// 验证角色值是否有效
		var newRole model.Role
		switch model.Role(req.Role) {
		case model.RoleAdmin, model.RoleOfficer, model.RoleGuest:
			newRole = model.Role(req.Role)
		default:
			c.BadRequest("无效的角色值")
			return fmt.Errorf("无效的角色值: %s", req.Role)
		}

		userService, err := service.GetUserServiceInstance()
		if err != nil {
			c.ServerError("系统错误", err.Error())
			return err
		}

		if err := userService.UpdateUserRole(uint(userID), newRole); err != nil {
			c.BadRequest(err.Error())
			return err
		}

		c.Success("用户角色更新成功")
		return nil
	})
}

// DeleteUser 删除用户
func (h *UserHandler) DeleteUser() gin.HandlerFunc {
	return h.Handle(func(c *Context) error {
		// 检查管理员权限
		if !c.RequireAdmin() {
			return fmt.Errorf("需要管理员权限")
		}

		// 获取用户ID参数
		userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.BadRequest("无效的用户ID", err.Error())
			return err
		}

		userService, err := service.GetUserServiceInstance()
		if err != nil {
			c.ServerError("系统错误", err.Error())
			return err
		}

		if err := userService.DeleteUser(uint(userID)); err != nil {
			c.ServerError("删除用户失败", err.Error())
			return err
		}

		c.Success("用户删除成功")
		return nil
	})
}

// CheckAuth 验证Token是否有效
func (h *UserHandler) CheckAuth() gin.HandlerFunc {
	return h.Handle(func(c *Context) error {
		var req dto.CheckAuthRequest
		if !c.BindRequest(&req) {
			return nil
		}

		valid, username, role := utils.CheckAuth(req.Token)

		if !valid {
			c.Data("令牌无效", dto.AuthResponse{
				StandardResponse: dto.StandardResponse{
					Code:    utils.StatusUnauthorized,
					Message: "令牌无效",
				},
				Valid: false,
			})
			return nil
		}

		userService, err := service.GetUserServiceInstance()
		if err != nil {
			c.ServerError("系统错误", err.Error())
			return err
		}

		user, err := userService.GetUserByUsername(username)
		if err != nil {
			c.Data("用户不存在", dto.AuthResponse{
				StandardResponse: dto.StandardResponse{
					Code:    utils.StatusUnauthorized,
					Message: "用户不存在",
				},
				Valid: false,
			})
			return err
		}

		// 准备响应数据
		userResp := dto.UserResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     role,
		}

		c.Data("令牌有效", dto.AuthResponse{
			StandardResponse: dto.StandardResponse{
				Code:    utils.StatusSuccess,
				Message: "令牌有效",
				Data:    map[string]interface{}{"user": userResp},
			},
			Valid: true,
		})
		return nil
	})
}

// UpdateUserID 管理员更新用户名
func (h *UserHandler) UpdateUserID() gin.HandlerFunc {
	return h.Handle(func(c *Context) error {
		// 检查管理员权限
		if !c.RequireAdmin() {
			return fmt.Errorf("需要管理员权限")
		}

		// 获取用户ID参数
		userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.BadRequest("无效的用户ID", err.Error())
			return err
		}

		var req dto.UpdateUsernameRequest
		if !c.BindRequest(&req) {
			return nil
		}

		userService, err := service.GetUserServiceInstance()
		if err != nil {
			c.ServerError("系统错误", err.Error())
			return err
		}

		if err := userService.UpdateUserID(uint(userID), req.Username); err != nil {
			c.BadRequest(err.Error())
			return err
		}

		c.Success("用户名更新成功")
		return nil
	})
}

// AdminChangePassword 管理员修改用户密码
func (h *UserHandler) AdminChangePassword() gin.HandlerFunc {
	return h.Handle(func(c *Context) error {
		// 检查管理员权限
		if !c.RequireAdmin() {
			return fmt.Errorf("需要管理员权限")
		}

		// 获取用户ID参数
		userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
		if err != nil {
			c.BadRequest("无效的用户ID", err.Error())
			return err
		}

		var req dto.AdminChangePasswordRequest
		if !c.BindRequest(&req) {
			return nil
		}

		userService, err := service.GetUserServiceInstance()
		if err != nil {
			c.ServerError("系统错误", err.Error())
			return err
		}

		if err := userService.AdminChangePassword(uint(userID), req.NewPassword); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.NotFound("用户不存在")
			} else {
				c.BadRequest(err.Error())
			}
			return err
		}

		c.Success("用户密码修改成功")
		return nil
	})
}