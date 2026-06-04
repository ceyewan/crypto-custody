package dto

// LoginRequest 用户登录请求结构体
type LoginRequest struct {
	Username   string `json:"username"`                    // 兼容旧字段
	Identifier string `json:"identifier"`                  // 登录标识
	Password   string `json:"password" binding:"required"` // 密码，必填
}

// RegisterRequest 用户注册请求结构体
type RegisterRequest struct {
	Username   string `json:"username"`                    // 兼容旧字段
	Identifier string `json:"identifier"`                  // 登录标识
	Nickname   string `json:"nickname"`                    // 昵称
	Password   string `json:"password" binding:"required"` // 密码，必填
	Email      string `json:"email"`                       // 历史兼容字段
}

// ChangePasswordRequest 修改密码请求结构体
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`       // 旧密码，必填
	NewPassword string `json:"newPassword" binding:"required,min=6"` // 新密码，必填，最少6个字符
}

// AdminChangePasswordRequest 管理员修改用户密码请求结构体
type AdminChangePasswordRequest struct {
	NewPassword string `json:"newPassword" binding:"required,min=6"` // 新密码，必填，最少6个字符
}

// UpdateRoleRequest 更新用户角色请求结构体
type UpdateRoleRequest struct {
	Role string `json:"role" binding:"required"` // 角色，必填
}

// UpdateUsernameRequest 更新用户名请求结构体
type UpdateUsernameRequest struct {
	Username string `json:"username" binding:"required"` // 用户名，必填
}

// UpdateUserStatusRequest 更新用户状态请求结构体
type UpdateUserStatusRequest struct {
	Status string `json:"status" binding:"required"` // 状态，必填
}

// CheckAuthRequest 验证令牌请求结构体
type CheckAuthRequest struct {
	Token string `json:"token" binding:"required"` // 令牌，必填
}

// UserResponse 用户信息响应结构体
type UserResponse struct {
	ID         uint   `json:"id"`         // 用户ID
	Username   string `json:"username"`   // 登录标识
	Identifier string `json:"identifier"` // 登录标识
	Nickname   string `json:"nickname"`   // 昵称
	Email      string `json:"email"`      // 历史邮箱字段
	Role       string `json:"role"`       // 角色
	Status     string `json:"status"`     // 状态
}

// LoginResponse 登录响应结构体
type LoginResponse struct {
	Token string       `json:"token"` // JWT令牌
	User  UserResponse `json:"user"`  // 用户信息
}

// StandardResponse 标准API响应结构体
type StandardResponse struct {
	Code    int         `json:"code"`           // 状态码
	Message string      `json:"message"`        // 消息
	Data    interface{} `json:"data,omitempty"` // 数据，可选
}

// AuthResponse 身份验证响应结构体
type AuthResponse struct {
	StandardResponse
	Valid bool `json:"valid"` // 令牌是否有效
}
