package service

import "example/database"

type UserService struct {
	db *database.DB
}

// @inject
func NewUserService(db *database.DB) *UserService {
	return &UserService{db: db}
}
