package service

import (
	"errors"
	"fmt"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

var ErrEpicNotFound = errors.New("指定されたエピックが存在しません")

// EpicInput はエピックの作成・更新の入力。StartDate・DueDate・Color の nil は、作成では未設定、更新では今の値のまま
type EpicInput struct {
	Title     string
	SpaceID   int
	StartDate *string
	DueDate   *string
	Color     *string
}

type EpicService interface {
	GetEpics() ([]domain.Epic, error)
	CreateEpic(input EpicInput) error
	// UpdateEpic・DeleteEpic で子タスクが変わったときは、actorID のユーザーの変更として履歴に記録する
	UpdateEpic(id int, input EpicInput, actorID int) error
	DeleteEpic(id int, actorID int) error
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

func (s *epicService) CreateEpic(input EpicInput) error {
	if err := validateSpace(s.spaceRepo, input.SpaceID); err != nil {
		return err
	}
	epic, err := applyEpicInput(domain.Epic{}, input)
	if err != nil {
		return err
	}
	return s.epicRepo.Create(epic)
}

// UpdateEpic はエピックを更新する。スペースが変わった場合は子タスクも同じスペースへ移す
func (s *epicService) UpdateEpic(id int, input EpicInput, actorID int) error {
	spaceID := input.SpaceID
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

	updated, err := applyEpicInput(*epic, input)
	if err != nil {
		return err
	}
	if err := s.epicRepo.Update(id, updated); err != nil {
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
		if err := updateWithHistory(s.todoRepo, s.history.by(actorID), s.statuses, names, todo, moved); err != nil {
			return fmt.Errorf("failed to move todo %d to space %d: %w", todo.ID, spaceID, err)
		}
	}

	return nil
}

// DeleteEpic はエピックを削除し、紐づく子タスクはエピック未割り当てに戻す
func (s *epicService) DeleteEpic(id int, actorID int) error {
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
		if err := updateWithHistory(s.todoRepo, s.history.by(actorID), s.statuses, names, todo, unlinked); err != nil {
			return fmt.Errorf("failed to unlink todo %d from epic %d: %w", todo.ID, id, err)
		}
	}

	return nil
}

// applyEpicInput は epic に入力を反映したエピックを返す。日付・色は nil なら epic の値のまま。
// 開始日・期限日の形式や前後、色の形式が正しくなければ domain のエラーを返す
func applyEpicInput(epic domain.Epic, input EpicInput) (domain.Epic, error) {
	epic.Title = input.Title
	epic.SpaceID = input.SpaceID
	if input.StartDate != nil {
		epic.StartDate = *input.StartDate
	}
	if input.DueDate != nil {
		epic.DueDate = *input.DueDate
	}
	if err := domain.ValidateEpicDates(epic.StartDate, epic.DueDate); err != nil {
		return domain.Epic{}, err
	}
	if input.Color != nil {
		color, err := domain.NormalizeEpicColor(*input.Color)
		if err != nil {
			return domain.Epic{}, err
		}
		epic.Color = color
	}
	return epic, nil
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
