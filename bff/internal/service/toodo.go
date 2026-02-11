package service

import (
	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

type TodoService interface {
	GetTodos() ([]domain.Todo, error)
	CreateTodo(title string, deadline string, status string) error
	UpdateTodo(id int, title string, done bool, deadline string, status string) error
	DeleteTodo(id int) error
}

type todoService struct {
	repo repository.TodoRepository
}

func NewTodoService(r repository.TodoRepository) TodoService {
	return &todoService{repo: r}
}

func (s *todoService) GetTodos() ([]domain.Todo, error) {
	return s.repo.FindAll()
}

func (s *todoService) CreateTodo(title string, deadline string, status string) error {
	// statusが空の場合はデフォルト値を設定
	todoStatus := domain.Status(status)
	if status == "" {
		todoStatus = domain.StatusOpen
	}

	todo := domain.Todo{
		Title:    title,
		Done:     false,
		Deadline: deadline,
		Status:   todoStatus,
	}
	return s.repo.Create(todo)
}

func (s *todoService) UpdateTodo(id int, title string, done bool, deadline string, status string) error {
	todo := domain.Todo{
		Title:    title,
		Done:     done,
		Deadline: deadline,
		Status:   domain.Status(status),
	}
	return s.repo.Update(id, todo)
}

func (s *todoService) DeleteTodo(id int) error {
	return s.repo.Delete(id)
}
