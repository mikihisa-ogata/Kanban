package service

import (
	"errors"
	"fmt"
	"strings"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

var (
	ErrSpaceNotFound     = errors.New("指定されたスペースが存在しません")
	ErrInvalidSpaceKey   = fmt.Errorf("キーは英大文字で始まる英大文字・数字の 2〜10 文字にしてください（%s は使えません）", domain.UnassignedKeyPrefix)
	ErrSpaceKeyDuplicate = errors.New("このキーは他のスペースで使われています")
)

type SpaceService interface {
	GetSpaces() ([]domain.Space, error)
	// CreateSpace はスペースを作成する。key が空なら名前から重複しないキーを作る
	CreateSpace(title string, key string) error
	// UpdateSpace はスペースの名前とキーを変える
	UpdateSpace(id int, title string, key string) error
	// SuggestSpaceKey は名前から、既存のスペースと重複しないキーの候補を返す
	SuggestSpaceKey(title string) (string, error)
	// DeleteSpace はスペースを削除する。課題の変更は actorID のユーザーの変更として履歴に記録する
	DeleteSpace(id int, actorID int) error
	// GetStatuses はすべてのワークフローのステータス（未割り当ての SpaceID 0 を含む）を、スペース・並び順の順に返す
	GetStatuses() ([]domain.WorkflowStatus, error)
	// CreateStatus・RenameStatus・ReorderStatuses・DeleteStatus はスペースのワークフローを編集する（workflow.go）。
	// 未割り当てのワークフローは編集できない
	CreateStatus(spaceID int, name string, category string) (domain.WorkflowStatus, error)
	RenameStatus(id int, name string) (domain.WorkflowStatus, error)
	ReorderStatuses(spaceID int, ids []int) error
	// SetStatusWIPLimit はステータスの列の最大件数を変える。nil と 0 は制限なし
	SetStatusWIPLimit(id int, limit *int) (domain.WorkflowStatus, error)
	// DeleteStatus で課題を別のステータスへ移したときは、actorID のユーザーの変更として履歴に記録する
	DeleteStatus(id int, moveToID int, actorID int) error
}

type spaceService struct {
	spaceRepo repository.SpaceRepository
	epicRepo  repository.EpicRepository
	todoRepo  repository.TodoRepository
	history   historyRecorder
	statuses  statusResolver
}

func NewSpaceService(s repository.SpaceRepository, e repository.EpicRepository, t repository.TodoRepository, h repository.HistoryRepository, st repository.StatusRepository) SpaceService {
	return &spaceService{spaceRepo: s, epicRepo: e, todoRepo: t, history: historyRecorder{history: h, epics: e, spaces: s}, statuses: statusResolver{repo: st}}
}

func (s *spaceService) GetStatuses() ([]domain.WorkflowStatus, error) {
	spaces, err := s.spaceRepo.FindAll()
	if err != nil {
		return nil, err
	}
	// ステータスのないスペースがあれば既定のワークフローを作ってから返す
	for _, id := range append([]int{0}, spaceIDs(spaces)...) {
		if err := s.statuses.repo.EnsureDefaults(id); err != nil {
			return nil, err
		}
	}
	return s.statuses.repo.FindAll()
}

func spaceIDs(spaces []domain.Space) []int {
	ids := make([]int, 0, len(spaces))
	for _, space := range spaces {
		ids = append(ids, space.ID)
	}
	return ids
}

func (s *spaceService) GetSpaces() ([]domain.Space, error) {
	return s.spaceRepo.FindAll()
}

func (s *spaceService) CreateSpace(title string, key string) error {
	taken, err := s.takenSpaceKeys(0)
	if err != nil {
		return err
	}

	key = strings.ToUpper(strings.TrimSpace(key))
	if key == "" {
		key = domain.SuggestSpaceKey(title, taken)
	}
	if !domain.ValidSpaceKey(key) {
		return ErrInvalidSpaceKey
	}
	if taken[key] {
		return ErrSpaceKeyDuplicate
	}
	if err := s.spaceRepo.Create(domain.Space{Title: title, Key: key}); err != nil {
		return err
	}
	// 作ったスペースに既定のワークフロー（4 ステータス）を作る。キーは重複しないので、キーでスペースを探す
	spaces, err := s.spaceRepo.FindAll()
	if err != nil {
		return err
	}
	for _, space := range spaces {
		if space.Key == key {
			return s.statuses.repo.EnsureDefaults(space.ID)
		}
	}
	return nil
}

func (s *spaceService) UpdateSpace(id int, title string, key string) error {
	exists, err := spaceExists(s.spaceRepo, id)
	if err != nil {
		return err
	}
	if id == 0 || !exists {
		return ErrSpaceNotFound
	}

	// 自分の今のキーのままなら重複ではない
	taken, err := s.takenSpaceKeys(id)
	if err != nil {
		return err
	}
	key = strings.ToUpper(strings.TrimSpace(key))
	if !domain.ValidSpaceKey(key) {
		return ErrInvalidSpaceKey
	}
	if taken[key] {
		return ErrSpaceKeyDuplicate
	}
	return s.spaceRepo.Update(id, domain.Space{Title: title, Key: key})
}

func (s *spaceService) SuggestSpaceKey(title string) (string, error) {
	taken, err := s.takenSpaceKeys(0)
	if err != nil {
		return "", err
	}
	return domain.SuggestSpaceKey(title, taken), nil
}

// takenSpaceKeys は既存のスペースが使っているキーを返す。excludeID のスペースのキーは含めない（0 なら全スペース）
func (s *spaceService) takenSpaceKeys(excludeID int) (map[string]bool, error) {
	spaces, err := s.spaceRepo.FindAll()
	if err != nil {
		return nil, err
	}
	taken := map[string]bool{}
	for _, space := range spaces {
		if space.Key != "" && space.ID != excludeID {
			taken[space.Key] = true
		}
	}
	return taken, nil
}

// DeleteSpace はスペースを削除し、属するエピックとタスクはスペース未割り当てに戻す
// （エピックとタスクの紐付けは残る）
func (s *spaceService) DeleteSpace(id int, actorID int) error {
	// 履歴に削除したスペースの名前と、課題の今の課題キーを残すため、削除の前に読む
	names, err := s.history.names()
	if err != nil {
		return err
	}
	todos, err := s.todoRepo.FindAll()
	if err != nil {
		return err
	}
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

	for _, todo := range todos {
		if todo.SpaceID != id {
			continue
		}
		unlinked := todo
		unlinked.SpaceID = 0
		if err := updateWithHistory(s.todoRepo, s.history.by(actorID), s.statuses, names, todo, unlinked); err != nil {
			return fmt.Errorf("failed to unlink todo %d from space %d: %w", todo.ID, id, err)
		}
	}

	// 課題は未割り当てのワークフローへ移したので、スペースのステータスは消す
	return s.statuses.repo.DeleteBySpace(id)
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
