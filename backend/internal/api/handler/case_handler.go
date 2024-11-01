// internal/api/handler/case_handler.go
package handler

import (
	"crypto-custody/internal/service"

	"github.com/gin-gonic/gin"
)

type CaseHandler struct {
	caseService *service.CaseService
}

func NewCaseHandler(caseService *service.CaseService) *CaseHandler {
	return &CaseHandler{caseService: caseService}
}

func (h *CaseHandler) Create(c *gin.Context) {
	// todo
}

func (h *CaseHandler) List(c *gin.Context) {
	// todo
}

func (h *CaseHandler) Get(c *gin.Context) {
	// todo
}

func (h *CaseHandler) Update(c *gin.Context) {
	// todo
}

func (h *CaseHandler) Delete(c *gin.Context) {
	// todo
}

func (h *CaseHandler) AssignPermissions(c *gin.Context) {
	// todo
}

func (h *CaseHandler) RevokePermissions(c *gin.Context) {
	// todo
}
