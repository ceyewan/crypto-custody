// internal/api/handler/wallet_handler.go
package handler

import (
	"crypto-custody/internal/service"

	"github.com/gin-gonic/gin"
)

type WalletHandler struct {
	walletService *service.WalletService
}

func NewWalletHandler(walletService *service.WalletService) *WalletHandler {
	return &WalletHandler{walletService: walletService}
}

func (h *WalletHandler) Create(c *gin.Context) {
	// todo
}

func (h *WalletHandler) List(c *gin.Context) {
	// todo
}

func (h *WalletHandler) Get(c *gin.Context) {
	// todo
}

func (h *WalletHandler) Transfer(c *gin.Context) {
	// todo
}

func (h *WalletHandler) Delete(c *gin.Context) {
	// todo
}
