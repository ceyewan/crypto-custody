package handler

import (
	"net/http"
	"offline-server/storage"
	"offline-server/storage/model"
	"time"

	"github.com/gin-gonic/gin"
)

type seResponse struct {
	ID              uint      `json:"id"`
	SeID            string    `json:"se_id"`
	CPLC            string    `json:"cplc"`
	Status          string    `json:"status"`
	CustodyLocation string    `json:"custody_location"`
	Remark          string    `json:"remark"`
	RegisteredBy    string    `json:"registered_by"`
	LastUsedAt      any       `json:"last_used_at,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreateSeRequest 创建SE的请求结构
type CreateSeRequest struct {
	SeID            string `json:"se_id" binding:"required"`
	CPLC            string `json:"cplc" binding:"required"`
	CustodyLocation string `json:"custody_location"`
}

// CreateSe 创建新的安全芯片记录
func CreateSe(c *gin.Context) {
	var req CreateSeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":  400,
			"error": "无效的请求参数",
		})
		return
	}

	seStorage := storage.GetSeStorage()
	se, err := seStorage.CreateSe(req.SeID, req.CPLC, req.CustodyLocation, usernameFromContext(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":  500,
			"error": "创建安全芯片记录失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": seDTO(*se),
	})
}

// ListSe 查询安全芯片列表。
func ListSe(c *gin.Context) {
	seStorage := storage.GetSeStorage()
	ses, err := seStorage.GetAllSe()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":  500,
			"error": "查询安全芯片列表失败: " + err.Error(),
		})
		return
	}
	data := make([]seResponse, 0, len(ses))
	for _, se := range ses {
		data = append(data, seDTO(se))
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": data,
	})
}

func seDTO(se model.Se) seResponse {
	return seResponse{
		ID:              se.ID,
		SeID:            se.SeID,
		CPLC:            se.CPLC,
		Status:          string(se.Status),
		CustodyLocation: se.CustodyLocation,
		Remark:          se.CustodyLocation,
		RegisteredBy:    se.RegisteredBy,
		LastUsedAt:      nil,
		CreatedAt:       se.CreatedAt,
		UpdatedAt:       se.UpdatedAt,
	}
}
