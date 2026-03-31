package service

import "example/database"

type UserService struct {
	db *database.DB
}

// @Inject
func NewUserService(db *database.DB) *UserService {
	return &UserService{db: db}
}
