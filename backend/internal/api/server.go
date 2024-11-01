// internal/api/server.go
package api

import (
	"fmt"

	"crypto-custody/config"
	"crypto-custody/internal/api/handler"
	"crypto-custody/internal/api/router"
	"crypto-custody/internal/service"

	"github.com/gin-gonic/gin"
)

type Server struct {
	config  *config.Config
	engine  *gin.Engine
	handler *handler.Handler
}

func NewServer(cfg *config.Config, services *service.Services) *Server {
	engine := gin.Default()
	// 创建 handler 实例
	h := handler.NewHandler(services)
	return &Server{
		config:  cfg,
		engine:  engine,
		handler: h,
	}
}

func (s *Server) Run() error {
	router.Setup(s.engine, s.handler)
	return s.engine.Run(fmt.Sprintf(":%d", s.config.Server.Port))
}
