package service

import (
	"errors"
	"strings"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

var (
	ErrFilterNotFound      = errors.New("指定されたフィルターが存在しません")
	ErrFilterNameDuplicate = errors.New("同じ名前のフィルターが既にあります")
)

type FilterService interface {
	// GetFilters はユーザーの保存済みフィルターを名前順に返す
	GetFilters(ownerID int) ([]domain.SavedFilter, error)
	// CreateFilter はユーザーのフィルターを保存する。JQL の構文の誤りは位置付きの *JQLError
	CreateFilter(ownerID int, name, jql string, favourite bool) (domain.SavedFilter, error)
	// UpdateFilter はフィルターの名前と JQL を変える（上書き保存）。ほかのユーザーのフィルターは ErrFilterNotFound
	UpdateFilter(id, ownerID int, name, jql string) (domain.SavedFilter, error)
	// SetFavourite はフィルターのお気に入り（スター）を付ける・外す
	SetFavourite(id, ownerID int, favourite bool) (domain.SavedFilter, error)
	// DeleteFilter はフィルターを削除する
	DeleteFilter(id, ownerID int) error
}

type filterService struct {
	repo repository.FilterRepository
}

func NewFilterService(r repository.FilterRepository) FilterService {
	return &filterService{repo: r}
}

func (s *filterService) GetFilters(ownerID int) ([]domain.SavedFilter, error) {
	return s.repo.FindByOwner(ownerID)
}

func (s *filterService) CreateFilter(ownerID int, name, jql string, favourite bool) (domain.SavedFilter, error) {
	filter := domain.SavedFilter{OwnerID: ownerID, Favourite: favourite}
	if err := s.applyFields(&filter, name, jql); err != nil {
		return domain.SavedFilter{}, err
	}
	id, err := s.repo.Create(filter)
	if err != nil {
		return domain.SavedFilter{}, err
	}
	return s.find(id, ownerID)
}

func (s *filterService) UpdateFilter(id, ownerID int, name, jql string) (domain.SavedFilter, error) {
	filter, err := s.find(id, ownerID)
	if err != nil {
		return domain.SavedFilter{}, err
	}
	if err := s.applyFields(&filter, name, jql); err != nil {
		return domain.SavedFilter{}, err
	}
	if err := s.repo.Update(filter); err != nil {
		return domain.SavedFilter{}, err
	}
	return s.find(id, ownerID)
}

func (s *filterService) SetFavourite(id, ownerID int, favourite bool) (domain.SavedFilter, error) {
	filter, err := s.find(id, ownerID)
	if err != nil {
		return domain.SavedFilter{}, err
	}
	filter.Favourite = favourite
	if err := s.repo.Update(filter); err != nil {
		return domain.SavedFilter{}, err
	}
	return s.find(id, ownerID)
}

func (s *filterService) DeleteFilter(id, ownerID int) error {
	if _, err := s.find(id, ownerID); err != nil {
		return err
	}
	return s.repo.Delete(id)
}

// find はユーザーのフィルターを返す。ないか、ほかのユーザーのものなら ErrFilterNotFound
func (s *filterService) find(id, ownerID int) (domain.SavedFilter, error) {
	filter, err := s.repo.FindByID(id)
	if err != nil {
		return domain.SavedFilter{}, err
	}
	if filter == nil || filter.OwnerID != ownerID {
		return domain.SavedFilter{}, ErrFilterNotFound
	}
	return *filter, nil
}

// applyFields は名前と JQL を検証して filter に入れる。JQL は構文だけを確かめる（名前の解決は検索するときに行う。
// 保存したあとでステータスなどの名前が変わることもあるため）。同じユーザーに同じ名前（大文字・小文字は区別しない）があれば ErrFilterNameDuplicate
func (s *filterService) applyFields(filter *domain.SavedFilter, name, jql string) error {
	name, err := domain.NormalizeFilterName(name)
	if err != nil {
		return err
	}
	jql, err = domain.NormalizeFilterJQL(jql)
	if err != nil {
		return err
	}
	if _, err := parseJQL(jql); err != nil {
		return err
	}
	filters, err := s.repo.FindByOwner(filter.OwnerID)
	if err != nil {
		return err
	}
	for _, f := range filters {
		if f.ID != filter.ID && strings.EqualFold(f.Name, name) {
			return ErrFilterNameDuplicate
		}
	}
	filter.Name, filter.JQL = name, jql
	return nil
}

type IssueViewService interface {
	// RecordView はユーザーが課題を表示したことを記録する（「最近表示した課題」に出る）。課題がなければ ErrIssueNotFound
	RecordView(todoID, userID int) error
}

type issueViewService struct {
	views    repository.IssueViewRepository
	todoRepo repository.TodoRepository
}

func NewIssueViewService(v repository.IssueViewRepository, t repository.TodoRepository) IssueViewService {
	return &issueViewService{views: v, todoRepo: t}
}

func (s *issueViewService) RecordView(todoID, userID int) error {
	if err := requireTodo(s.todoRepo, todoID); err != nil {
		return err
	}
	return s.views.Record(userID, todoID)
}
