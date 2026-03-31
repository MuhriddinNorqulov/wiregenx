package database

import "example/config"

type DB struct {
	cfg *config.Config
}

// @inject
func NewDB(cfg *config.Config) (*DB, error) {
	return &DB{cfg: cfg}, nil
}
