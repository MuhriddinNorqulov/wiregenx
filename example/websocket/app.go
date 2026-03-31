package websocket

import (
	"example/config"
	"example/database"
)

type App struct {
	cfg *config.Config
	db  *database.DB
}

// @Application
func NewApp(cfg *config.Config, db *database.DB) *App {
	return &App{cfg: cfg, db: db}
}
