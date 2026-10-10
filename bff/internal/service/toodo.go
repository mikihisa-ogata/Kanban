package service

import (
	"errors"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

var ErrEpicSpaceMismatch = errors.New("エピックとタスクのスペースが一致しません")

var ErrIssueNotFound = errors.New("課題が見つかりません")

var ErrInvalidIssueType = errors.New("課題タイプは Task・Bug・Story のいずれかにしてください")

type TodoService interface {
	GetTodos() ([]domain.Todo, error)
	// GetTodoByKey は課題キー（例: KAN-12）のタスクを返す。形式が正しくないか存在しなければ ErrIssueNotFound を返す
	GetTodoByKey(key string) (domain.Todo, error)
	// CreateTodo はタスクを作成し、採番した課題キーを含む作成後のタスクを返す。課題タイプが空なら Task にする
	CreateTodo(title string, deadline string, status string, issueType string, epicID int, description string, spaceID int) (domain.Todo, error)
	// UpdateTodo はタスクを更新し、更新後のタスクを返す（スペースを変えると課題キーが変わる）。課題タイプが空なら今のタイプのままにする
	UpdateTodo(id int, title string, done bool, deadline string, status string, issueType string, epicID int, description string, spaceID int) (domain.Todo, error)
	DeleteTodo(id int) error
	MoveTodo(id int, beforeID int) error
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

func (s *todoService) GetTodoByKey(key string) (domain.Todo, error) {
	prefix, number, ok := domain.ParseIssueKey(key)
	if !ok {
		return domain.Todo{}, ErrIssueNotFound
	}
	todo, err := s.repo.FindByKey(prefix, number)
	if err != nil {
		return domain.Todo{}, err
	}
	if todo == nil {
		return domain.Todo{}, ErrIssueNotFound
	}
	return *todo, nil
}

func (s *todoService) CreateTodo(title string, deadline string, status string, issueType string, epicID int, description string, spaceID int) (domain.Todo, error) {
	if err := s.validateEpicAndSpace(epicID, spaceID); err != nil {
		return domain.Todo{}, err
	}
	todoType := domain.IssueType(issueType)
	if issueType == "" {
		todoType = domain.IssueTypeTask
	}
	if !domain.ValidIssueType(todoType) {
		return domain.Todo{}, ErrInvalidIssueType
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
		IssueType:   todoType,
		EpicID:      epicID,
		Description: description,
		SpaceID:     spaceID,
	}
	id, err := s.repo.Create(todo)
	if err != nil {
		return domain.Todo{}, err
	}
	return s.repo.FindByID(id)
}

func (s *todoService) UpdateTodo(id int, title string, done bool, deadline string, status string, issueType string, epicID int, description string, spaceID int) (domain.Todo, error) {
	if err := s.validateEpicAndSpace(epicID, spaceID); err != nil {
		return domain.Todo{}, err
	}
	// 課題タイプを送らない呼び出し（ドラッグでの移動など）でタイプが変わらないよう、空なら今のタイプを使う
	todoType := domain.IssueType(issueType)
	if issueType == "" {
		current, err := s.repo.FindByID(id)
		if err != nil {
			return domain.Todo{}, err
		}
		todoType = current.IssueType
	}
	if !domain.ValidIssueType(todoType) {
		return domain.Todo{}, ErrInvalidIssueType
	}

	todo := domain.Todo{
		Title:       title,
		Done:        done,
		Deadline:    deadline,
		Status:      domain.Status(status),
		IssueType:   todoType,
		EpicID:      epicID,
		Description: description,
		SpaceID:     spaceID,
	}
	if err := s.repo.Update(id, todo); err != nil {
		return domain.Todo{}, err
	}
	return s.repo.FindByID(id)
}

func (s *todoService) DeleteTodo(id int) error {
	return s.repo.Delete(id)
}

// MoveTodo はタスクを beforeID のタスクの直前へ移動する。beforeID が 0 の場合は末尾へ移動する
func (s *todoService) MoveTodo(id int, beforeID int) error {
	return s.repo.Move(id, beforeID)
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
