// internal/api/handler/handler.go
package handler

import (
	"crypto-custody/internal/service"
)

type Handler struct {
	User   *UserHandler
	Case   *CaseHandler
	Wallet *WalletHandler
	Log    *LogHandler
}

func NewHandler(services *service.Services) *Handler {
	return &Handler{
		User:   NewUserHandler(services.User),
		Case:   NewCaseHandler(services.Case),
		Wallet: NewWalletHandler(services.Wallet),
		Log:    NewLogHandler(services.Log),
	}
}


