package http

import (
	"example/config"
	"example/service"
)

type App struct {
	cfg     *config.Config
	userSvc *service.UserService
}

// @Application("http")
func NewApp(cfg *config.Config, userSvc *service.UserService) *App {
	return &App{cfg: cfg, userSvc: userSvc}
}
