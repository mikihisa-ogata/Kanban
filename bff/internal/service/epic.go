package service

import (
	"fmt"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

type EpicService interface {
	GetEpics() ([]domain.Epic, error)
	CreateEpic(title string) error
	DeleteEpic(id int) error
}

type epicService struct {
	epicRepo repository.EpicRepository
	todoRepo repository.TodoRepository
}

func NewEpicService(e repository.EpicRepository, t repository.TodoRepository) EpicService {
	return &epicService{epicRepo: e, todoRepo: t}
}

func (s *epicService) GetEpics() ([]domain.Epic, error) {
	return s.epicRepo.FindAll()
}

func (s *epicService) CreateEpic(title string) error {
	return s.epicRepo.Create(domain.Epic{Title: title})
}

// DeleteEpic はエピックを削除し、紐づく子タスクはエピック未割り当てに戻す
func (s *epicService) DeleteEpic(id int) error {
	if err := s.epicRepo.Delete(id); err != nil {
		return err
	}

	todos, err := s.todoRepo.FindAll()
	if err != nil {
		return err
	}

	for _, todo := range todos {
		if todo.EpicID != id {
			continue
		}
		todo.EpicID = 0
		if err := s.todoRepo.Update(todo.ID, todo); err != nil {
			return fmt.Errorf("failed to unlink todo %d from epic %d: %w", todo.ID, id, err)
		}
	}

	return nil
}

// epicExists は指定IDのエピックが存在するかを返す（0 は未割り当てとして常に有効）
func epicExists(repo repository.EpicRepository, id int) (bool, error) {
	if id == 0 {
		return true, nil
	}

	epics, err := repo.FindAll()
	if err != nil {
		return false, err
	}

	for _, epic := range epics {
		if epic.ID == id {
			return true, nil
		}
	}

	return false, nil
}
