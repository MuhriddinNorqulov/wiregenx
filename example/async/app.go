package async

import (
	"example/config"
	"example/service"
)

type App struct {
	cfg     *config.Config
	userSvc *service.UserService
}

// @Application("async")
func NewApp(cfg *config.Config, userSvc *service.UserService) *App {
	return &App{cfg: cfg, userSvc: userSvc}
}
