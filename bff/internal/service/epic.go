package service

import (
	"errors"
	"fmt"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

var ErrEpicNotFound = errors.New("指定されたエピックが存在しません")

type EpicService interface {
	GetEpics() ([]domain.Epic, error)
	CreateEpic(title string, spaceID int) error
	UpdateEpic(id int, title string, spaceID int) error
	DeleteEpic(id int) error
}

type epicService struct {
	epicRepo  repository.EpicRepository
	todoRepo  repository.TodoRepository
	spaceRepo repository.SpaceRepository
	history   historyRecorder
	statuses  statusResolver
}

func NewEpicService(e repository.EpicRepository, t repository.TodoRepository, s repository.SpaceRepository, h repository.HistoryRepository, st repository.StatusRepository) EpicService {
	return &epicService{epicRepo: e, todoRepo: t, spaceRepo: s, history: historyRecorder{history: h, epics: e, spaces: s}, statuses: statusResolver{repo: st}}
}

func (s *epicService) GetEpics() ([]domain.Epic, error) {
	return s.epicRepo.FindAll()
}

func (s *epicService) CreateEpic(title string, spaceID int) error {
	if err := validateSpace(s.spaceRepo, spaceID); err != nil {
		return err
	}
	return s.epicRepo.Create(domain.Epic{Title: title, SpaceID: spaceID})
}

// UpdateEpic はエピックを更新する。スペースが変わった場合は子タスクも同じスペースへ移す
func (s *epicService) UpdateEpic(id int, title string, spaceID int) error {
	if err := validateSpace(s.spaceRepo, spaceID); err != nil {
		return err
	}

	epic, err := findEpic(s.epicRepo, id)
	if err != nil {
		return err
	}
	if epic == nil {
		return ErrEpicNotFound
	}

	if err := s.epicRepo.Update(id, domain.Epic{Title: title, SpaceID: spaceID}); err != nil {
		return err
	}
	if epic.SpaceID == spaceID {
		return nil
	}

	names, err := s.history.names()
	if err != nil {
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
		moved := todo
		moved.SpaceID = spaceID
		if err := updateWithHistory(s.todoRepo, s.history, s.statuses, names, todo, moved); err != nil {
			return fmt.Errorf("failed to move todo %d to space %d: %w", todo.ID, spaceID, err)
		}
	}

	return nil
}

// DeleteEpic はエピックを削除し、紐づく子タスクはエピック未割り当てに戻す
func (s *epicService) DeleteEpic(id int) error {
	// 履歴に削除したエピックの名前を残すため、削除の前に読む
	names, err := s.history.names()
	if err != nil {
		return err
	}
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
		unlinked := todo
		unlinked.EpicID = 0
		if err := updateWithHistory(s.todoRepo, s.history, s.statuses, names, todo, unlinked); err != nil {
			return fmt.Errorf("failed to unlink todo %d from epic %d: %w", todo.ID, id, err)
		}
	}

	return nil
}

// findEpic は指定IDのエピックを返す（存在しない場合は nil）
func findEpic(repo repository.EpicRepository, id int) (*domain.Epic, error) {
	epics, err := repo.FindAll()
	if err != nil {
		return nil, err
	}

	for _, epic := range epics {
		if epic.ID == id {
			return &epic, nil
		}
	}

	return nil, nil
}
