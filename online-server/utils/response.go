package utils

import (
	"errors"
	"net/http"
	"online-server/dto"
	"online-server/model"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// 错误类型常量
const (
	ErrorBadRequest          = "请求参数不正确"
	ErrorUnauthorized        = "未授权访问"
	ErrorForbidden           = "权限不足"
	ErrorNotFound            = "资源不存在"
	ErrorInternalServerError = "系统错误"
	ErrorUserNotFound        = "未找到用户信息"
	ErrorInvalidID           = "无效的ID参数"
	ErrorRecordNotFound      = "记录不存在"
)

// ResponseWithError 响应错误信息
func ResponseWithError(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, dto.StandardResponse{
		Code:    statusCode,
		Message: message,
	})
}

// ResponseWithData 响应成功数据
func ResponseWithData(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, dto.StandardResponse{
		Code:    http.StatusOK,
		Message: message,
		Data:    data,
	})
}

// ResponseWithSuccess 响应成功消息（无数据）
func ResponseWithSuccess(c *gin.Context, message string) {
	c.JSON(http.StatusOK, dto.StandardResponse{
		Code:    http.StatusOK,
		Message: message,
	})
}

// BindJSON 绑定JSON请求并处理错误
func BindJSON(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		ResponseWithError(c, http.StatusBadRequest, ErrorBadRequest+": "+err.Error())
		return false
	}
	return true
}

// GetCurrentUser 获取当前登录用户
func GetCurrentUser(c *gin.Context) (*model.User, bool) {
	user, exists := c.Get("user")
	if !exists {
		ResponseWithError(c, http.StatusUnauthorized, ErrorUserNotFound)
		return nil, false
	}
	return user.(*model.User), true
}

// CheckAdminRole 检查用户是否具有管理员权限
func CheckAdminRole(c *gin.Context) bool {
	user, ok := GetCurrentUser(c)
	if !ok {
		return false
	}
	
	if user.Role != model.RoleAdmin {
		ResponseWithError(c, http.StatusForbidden, ErrorForbidden+", 需要管理员权限")
		return false
	}
	
	return true
}

// ParseUintParam 解析URL参数为uint
func ParseUintParam(c *gin.Context, param string) (uint, bool) {
	idStr := c.Param(param)
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		ResponseWithError(c, http.StatusBadRequest, ErrorInvalidID)
		return 0, false
	}
	return uint(id), true
}

// HandleServiceError 处理服务层的常见错误
func HandleServiceError(c *gin.Context, err error, notFoundMsg string) bool {
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ResponseWithError(c, http.StatusNotFound, notFoundMsg)
		} else {
			ResponseWithError(c, http.StatusInternalServerError, ErrorInternalServerError+": "+err.Error())
		}
		return true
	}
	return false
}

// HandleServiceInitError 处理服务初始化错误
func HandleServiceInitError(c *gin.Context, err error) bool {
	if err != nil {
		ResponseWithError(c, http.StatusInternalServerError, ErrorInternalServerError+": "+err.Error())
		return true
	}
	return false
}