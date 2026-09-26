package service

import (
	"errors"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

var ErrEpicNotFound = errors.New("指定されたエピックが存在しません")

type TodoService interface {
	GetTodos() ([]domain.Todo, error)
	CreateTodo(title string, deadline string, status string, epicID int, description string) error
	UpdateTodo(id int, title string, done bool, deadline string, status string, epicID int, description string) error
	DeleteTodo(id int) error
}

type todoService struct {
	repo     repository.TodoRepository
	epicRepo repository.EpicRepository
}

func NewTodoService(r repository.TodoRepository, e repository.EpicRepository) TodoService {
	return &todoService{repo: r, epicRepo: e}
}

func (s *todoService) GetTodos() ([]domain.Todo, error) {
	return s.repo.FindAll()
}

func (s *todoService) CreateTodo(title string, deadline string, status string, epicID int, description string) error {
	if err := s.validateEpic(epicID); err != nil {
		return err
	}

	// statusが空の場合はデフォルト値を設定
	todoStatus := domain.Status(status)
	if status == "" {
		todoStatus = domain.StatusOpen
	}

	todo := domain.Todo{
		Title:       title,
		Done:        false,
		Deadline:    deadline,
		Status:      todoStatus,
		EpicID:      epicID,
		Description: description,
	}
	return s.repo.Create(todo)
}

func (s *todoService) UpdateTodo(id int, title string, done bool, deadline string, status string, epicID int, description string) error {
	if err := s.validateEpic(epicID); err != nil {
		return err
	}

	todo := domain.Todo{
		Title:       title,
		Done:        done,
		Deadline:    deadline,
		Status:      domain.Status(status),
		EpicID:      epicID,
		Description: description,
	}
	return s.repo.Update(id, todo)
}

func (s *todoService) DeleteTodo(id int) error {
	return s.repo.Delete(id)
}

func (s *todoService) validateEpic(epicID int) error {
	exists, err := epicExists(s.epicRepo, epicID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrEpicNotFound
	}
	return nil
}
