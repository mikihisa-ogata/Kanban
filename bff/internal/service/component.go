package service

import (
	"errors"
	"sort"
	"strings"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

var (
	ErrComponentNotFound      = errors.New("指定されたコンポーネントが存在しません")
	ErrUnassignedComponent    = errors.New("スペース未割り当ての課題にはコンポーネントを作れません")
	ErrComponentNameDuplicate = errors.New("同じ名前のコンポーネントが既にあります")
	ErrComponentOtherSpace    = errors.New("課題と同じスペースのコンポーネントを選んでください")
)

type ComponentService interface {
	// GetComponents はすべてのコンポーネントを、スペース・名前順に返す
	GetComponents() ([]domain.Component, error)
	// CreateComponent はスペースにコンポーネントを作る。defaultAssigneeID は既定の担当者（0 はなし）
	CreateComponent(spaceID int, name, description string, defaultAssigneeID int) (domain.Component, error)
	// UpdateComponent はコンポーネントの名前・説明・既定の担当者を変える
	UpdateComponent(id int, name, description string, defaultAssigneeID int) (domain.Component, error)
	// DeleteComponent はコンポーネントを削除する。課題からも外れる
	DeleteComponent(id int) error
	// SetTodoComponents は課題のコンポーネントを componentIDs（課題と同じスペースのコンポーネント）に置き換え、変更後の課題を返す。
	// 課題の担当者が空なら、新しく付けたコンポーネントのうち ID 順で最初に既定の担当者を持つものの、その人を担当者にする（Jira と同じ）
	SetTodoComponents(todoID int, componentIDs []int, actorID int) (domain.Todo, error)
}

type componentService struct {
	repo      repository.ComponentRepository
	todoRepo  repository.TodoRepository
	spaceRepo repository.SpaceRepository
	userRepo  repository.UserRepository
	watchers  repository.WatcherRepository
	history   historyRecorder
	statuses  statusResolver
}

func NewComponentService(r repository.ComponentRepository, t repository.TodoRepository, e repository.EpicRepository, s repository.SpaceRepository, u repository.UserRepository, w repository.WatcherRepository, h repository.HistoryRepository, st repository.StatusRepository) ComponentService {
	return &componentService{
		repo: r, todoRepo: t, spaceRepo: s, userRepo: u, watchers: w,
		history:  historyRecorder{history: h, epics: e, spaces: s},
		statuses: statusResolver{repo: st},
	}
}

func (s *componentService) GetComponents() ([]domain.Component, error) {
	return s.repo.FindAll()
}

func (s *componentService) CreateComponent(spaceID int, name, description string, defaultAssigneeID int) (domain.Component, error) {
	if spaceID == 0 {
		return domain.Component{}, ErrUnassignedComponent
	}
	exists, err := spaceExists(s.spaceRepo, spaceID)
	if err != nil {
		return domain.Component{}, err
	}
	if !exists {
		return domain.Component{}, ErrSpaceNotFound
	}
	component := domain.Component{SpaceID: spaceID}
	if err := s.applyFields(&component, name, description, defaultAssigneeID); err != nil {
		return domain.Component{}, err
	}
	id, err := s.repo.Create(component)
	if err != nil {
		return domain.Component{}, err
	}
	return s.find(id)
}

func (s *componentService) UpdateComponent(id int, name, description string, defaultAssigneeID int) (domain.Component, error) {
	component, err := s.find(id)
	if err != nil {
		return domain.Component{}, err
	}
	if err := s.applyFields(&component, name, description, defaultAssigneeID); err != nil {
		return domain.Component{}, err
	}
	if err := s.repo.Update(component); err != nil {
		return domain.Component{}, err
	}
	return s.find(id)
}

func (s *componentService) DeleteComponent(id int) error {
	if _, err := s.find(id); err != nil {
		return err
	}
	return s.repo.Delete(id)
}

func (s *componentService) SetTodoComponents(todoID int, componentIDs []int, actorID int) (domain.Todo, error) {
	if err := requireTodo(s.todoRepo, todoID); err != nil {
		return domain.Todo{}, err
	}
	todo, err := s.todoRepo.FindByID(todoID)
	if err != nil {
		return domain.Todo{}, err
	}
	ids := []int{}
	assigneeID := 0
	for _, id := range componentIDs {
		if containsInt(ids, id) {
			continue
		}
		component, err := s.find(id)
		if err != nil {
			return domain.Todo{}, err
		}
		if component.SpaceID != todo.SpaceID {
			return domain.Todo{}, ErrComponentOtherSpace
		}
		ids = append(ids, id)
	}
	sort.Ints(ids)
	if todo.AssigneeID == 0 {
		for _, id := range ids {
			if containsInt(todo.ComponentIDs, id) {
				continue
			}
			component, err := s.find(id)
			if err != nil {
				return domain.Todo{}, err
			}
			if component.DefaultAssignee != nil {
				assigneeID = component.DefaultAssigneeID
				break
			}
		}
	}
	if err := s.repo.SetTodoComponents(todoID, ids); err != nil {
		return domain.Todo{}, err
	}
	if assigneeID != 0 {
		before, err := s.todoRepo.FindByID(todoID)
		if err != nil {
			return domain.Todo{}, err
		}
		after := before
		after.AssigneeID = assigneeID
		// 担当者を変えたときと同じく、新しい担当者をウォッチャーにしてから履歴に残す（割り当てられたことが通知される）
		if err := s.watchers.Add(todoID, assigneeID); err != nil {
			return domain.Todo{}, err
		}
		names, err := s.history.names()
		if err != nil {
			return domain.Todo{}, err
		}
		if err := updateWithHistory(s.todoRepo, s.history.by(actorID), s.statuses, names, before, after); err != nil {
			return domain.Todo{}, err
		}
	}
	return s.todoRepo.FindByID(todoID)
}

func (s *componentService) find(id int) (domain.Component, error) {
	component, err := s.repo.FindByID(id)
	if err != nil {
		return domain.Component{}, err
	}
	if component == nil {
		return domain.Component{}, ErrComponentNotFound
	}
	return *component, nil
}

// applyFields は名前・説明・既定の担当者を検証して component に入れる。同じスペースに同じ名前（大文字・小文字は区別しない）があれば ErrComponentNameDuplicate、
// 既定の担当者が存在しなければ ErrUserNotFound
func (s *componentService) applyFields(component *domain.Component, name, description string, defaultAssigneeID int) error {
	name, err := domain.NormalizeComponentName(name)
	if err != nil {
		return err
	}
	description, err = domain.NormalizeComponentDescription(description)
	if err != nil {
		return err
	}
	if defaultAssigneeID < 0 {
		return ErrUserNotFound
	}
	if defaultAssigneeID != 0 {
		user, err := s.userRepo.FindByID(defaultAssigneeID)
		if err != nil {
			return err
		}
		if user == nil {
			return ErrUserNotFound
		}
	}
	components, err := s.repo.FindAll()
	if err != nil {
		return err
	}
	for _, c := range components {
		if c.SpaceID == component.SpaceID && c.ID != component.ID && strings.EqualFold(c.Name, name) {
			return ErrComponentNameDuplicate
		}
	}
	component.Name, component.Description, component.DefaultAssigneeID = name, description, defaultAssigneeID
	return nil
}
