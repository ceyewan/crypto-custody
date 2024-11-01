package handler

import (
	"crypto-custody/internal/service"

	"github.com/gin-gonic/gin"
)

type LogHandler struct {
	logService *service.LogService
}

func NewLogHandler(logService *service.LogService) *LogHandler {
	return &LogHandler{logService: logService}
}

func (h *LogHandler) List(c *gin.Context) {
	// todo
}

func (h *LogHandler) Get(c *gin.Context) {
	// todo
}

func (h *LogHandler) Delete(c *gin.Context) {
	// todo
}

func (h *LogHandler) ListSystemLogs(c *gin.Context) {
	// todo
}

func (h *LogHandler) ListCaseLogs(c *gin.Context) {
	// todo
}

func (h *LogHandler) ListTransactionLogs(c *gin.Context) {
	// todo
}
