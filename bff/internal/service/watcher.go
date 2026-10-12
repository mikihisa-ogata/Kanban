package service

import (
	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

type WatcherService interface {
	// GetWatchers は課題のウォッチャーを表示名の順に返す。課題がなければ ErrIssueNotFound を返す
	GetWatchers(todoID int) ([]domain.UserSummary, error)
	// Watch はユーザーを課題のウォッチャーにし、ウォッチャーを返す。課題がなければ ErrIssueNotFound を返す
	Watch(todoID int, userID int) ([]domain.UserSummary, error)
	// Unwatch はユーザーを課題のウォッチャーから外し、ウォッチャーを返す。課題がなければ ErrIssueNotFound を返す
	Unwatch(todoID int, userID int) ([]domain.UserSummary, error)
}

type watcherService struct {
	watchers repository.WatcherRepository
	todoRepo repository.TodoRepository
}

func NewWatcherService(w repository.WatcherRepository, t repository.TodoRepository) WatcherService {
	return &watcherService{watchers: w, todoRepo: t}
}

func (s *watcherService) GetWatchers(todoID int) ([]domain.UserSummary, error) {
	if err := requireTodo(s.todoRepo, todoID); err != nil {
		return nil, err
	}
	return s.watchers.FindByTodo(todoID)
}

func (s *watcherService) Watch(todoID int, userID int) ([]domain.UserSummary, error) {
	if err := requireTodo(s.todoRepo, todoID); err != nil {
		return nil, err
	}
	if err := s.watchers.Add(todoID, userID); err != nil {
		return nil, err
	}
	return s.watchers.FindByTodo(todoID)
}

func (s *watcherService) Unwatch(todoID int, userID int) ([]domain.UserSummary, error) {
	if err := requireTodo(s.todoRepo, todoID); err != nil {
		return nil, err
	}
	if err := s.watchers.Remove(todoID, userID); err != nil {
		return nil, err
	}
	return s.watchers.FindByTodo(todoID)
}
