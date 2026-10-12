package service

import (
	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

type UserService interface {
	// GetUsers はすべてのユーザーを表示名の順に返す（担当者・報告者を選ぶ候補）
	GetUsers() ([]domain.User, error)
}

type userService struct {
	userRepo repository.UserRepository
}

func NewUserService(u repository.UserRepository) UserService {
	return &userService{userRepo: u}
}

func (s *userService) GetUsers() ([]domain.User, error) {
	return s.userRepo.FindAll()
}
