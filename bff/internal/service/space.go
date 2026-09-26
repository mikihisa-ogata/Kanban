package service

import (
	"errors"
	"fmt"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

var ErrSpaceNotFound = errors.New("指定されたスペースが存在しません")

type SpaceService interface {
	GetSpaces() ([]domain.Space, error)
	CreateSpace(title string) error
	DeleteSpace(id int) error
}

type spaceService struct {
	spaceRepo repository.SpaceRepository
	epicRepo  repository.EpicRepository
	todoRepo  repository.TodoRepository
}

func NewSpaceService(s repository.SpaceRepository, e repository.EpicRepository, t repository.TodoRepository) SpaceService {
	return &spaceService{spaceRepo: s, epicRepo: e, todoRepo: t}
}

func (s *spaceService) GetSpaces() ([]domain.Space, error) {
	return s.spaceRepo.FindAll()
}

func (s *spaceService) CreateSpace(title string) error {
	return s.spaceRepo.Create(domain.Space{Title: title})
}

// DeleteSpace はスペースを削除し、属するエピックとタスクはスペース未割り当てに戻す
// （エピックとタスクの紐付けは残る）
func (s *spaceService) DeleteSpace(id int) error {
	if err := s.spaceRepo.Delete(id); err != nil {
		return err
	}

	epics, err := s.epicRepo.FindAll()
	if err != nil {
		return err
	}
	for _, epic := range epics {
		if epic.SpaceID != id {
			continue
		}
		epic.SpaceID = 0
		if err := s.epicRepo.Update(epic.ID, epic); err != nil {
			return fmt.Errorf("failed to unlink epic %d from space %d: %w", epic.ID, id, err)
		}
	}

	todos, err := s.todoRepo.FindAll()
	if err != nil {
		return err
	}
	for _, todo := range todos {
		if todo.SpaceID != id {
			continue
		}
		todo.SpaceID = 0
		if err := s.todoRepo.Update(todo.ID, todo); err != nil {
			return fmt.Errorf("failed to unlink todo %d from space %d: %w", todo.ID, id, err)
		}
	}

	return nil
}

// spaceExists は指定IDのスペースが存在するかを返す（0 は未割り当てとして常に有効）
func spaceExists(repo repository.SpaceRepository, id int) (bool, error) {
	if id == 0 {
		return true, nil
	}

	spaces, err := repo.FindAll()
	if err != nil {
		return false, err
	}

	for _, space := range spaces {
		if space.ID == id {
			return true, nil
		}
	}

	return false, nil
}

func validateSpace(repo repository.SpaceRepository, id int) error {
	exists, err := spaceExists(repo, id)
	if err != nil {
		return err
	}
	if !exists {
		return ErrSpaceNotFound
	}
	return nil
}
