package service

import (
	"errors"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

var ErrEpicSpaceMismatch = errors.New("エピックとタスクのスペースが一致しません")

type TodoService interface {
	GetTodos() ([]domain.Todo, error)
	CreateTodo(title string, deadline string, status string, epicID int, description string, spaceID int) error
	UpdateTodo(id int, title string, done bool, deadline string, status string, epicID int, description string, spaceID int) error
	DeleteTodo(id int) error
}

type todoService struct {
	repo      repository.TodoRepository
	epicRepo  repository.EpicRepository
	spaceRepo repository.SpaceRepository
}

func NewTodoService(r repository.TodoRepository, e repository.EpicRepository, s repository.SpaceRepository) TodoService {
	return &todoService{repo: r, epicRepo: e, spaceRepo: s}
}

func (s *todoService) GetTodos() ([]domain.Todo, error) {
	return s.repo.FindAll()
}

func (s *todoService) CreateTodo(title string, deadline string, status string, epicID int, description string, spaceID int) error {
	if err := s.validateEpicAndSpace(epicID, spaceID); err != nil {
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
		SpaceID:     spaceID,
	}
	return s.repo.Create(todo)
}

func (s *todoService) UpdateTodo(id int, title string, done bool, deadline string, status string, epicID int, description string, spaceID int) error {
	if err := s.validateEpicAndSpace(epicID, spaceID); err != nil {
		return err
	}

	todo := domain.Todo{
		Title:       title,
		Done:        done,
		Deadline:    deadline,
		Status:      domain.Status(status),
		EpicID:      epicID,
		Description: description,
		SpaceID:     spaceID,
	}
	return s.repo.Update(id, todo)
}

func (s *todoService) DeleteTodo(id int) error {
	return s.repo.Delete(id)
}

// validateEpicAndSpace はスペースとエピックが存在し、エピックがタスクと同じスペースに属することを確かめる
func (s *todoService) validateEpicAndSpace(epicID int, spaceID int) error {
	if err := validateSpace(s.spaceRepo, spaceID); err != nil {
		return err
	}
	if epicID == 0 {
		return nil
	}

	epic, err := findEpic(s.epicRepo, epicID)
	if err != nil {
		return err
	}
	if epic == nil {
		return ErrEpicNotFound
	}
	if epic.SpaceID != spaceID {
		return ErrEpicSpaceMismatch
	}
	return nil
}
