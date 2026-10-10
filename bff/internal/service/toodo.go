package service

import (
	"errors"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

var ErrEpicSpaceMismatch = errors.New("エピックとタスクのスペースが一致しません")

var ErrIssueNotFound = errors.New("課題が見つかりません")

var ErrInvalidIssueType = errors.New("課題タイプは Task・Bug・Story のいずれかにしてください")

var ErrInvalidPriority = errors.New("優先度は Highest・High・Medium・Low・Lowest のいずれかにしてください")

type TodoService interface {
	GetTodos() ([]domain.Todo, error)
	// GetTodoByKey は課題キー（例: KAN-12）のタスクを返す。形式が正しくないか存在しなければ ErrIssueNotFound を返す
	GetTodoByKey(key string) (domain.Todo, error)
	// CreateTodo はタスクを作成し、採番した課題キーを含む作成後のタスクを返す。課題タイプが空なら Task、優先度が空なら Medium にする
	CreateTodo(title string, deadline string, status string, issueType string, priority string, epicID int, description string, spaceID int) (domain.Todo, error)
	// UpdateTodo はタスクを更新し、更新後のタスクを返す（スペースを変えると課題キーが変わる）。
	// 課題タイプ・優先度が空なら今の値のままにする
	UpdateTodo(id int, title string, done bool, deadline string, status string, issueType string, priority string, epicID int, description string, spaceID int) (domain.Todo, error)
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

func (s *todoService) CreateTodo(title string, deadline string, status string, issueType string, priority string, epicID int, description string, spaceID int) (domain.Todo, error) {
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
	todoPriority := domain.Priority(priority)
	if priority == "" {
		todoPriority = domain.PriorityMedium
	}
	if !domain.ValidPriority(todoPriority) {
		return domain.Todo{}, ErrInvalidPriority
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
		Priority:    todoPriority,
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

func (s *todoService) UpdateTodo(id int, title string, done bool, deadline string, status string, issueType string, priority string, epicID int, description string, spaceID int) (domain.Todo, error) {
	if err := s.validateEpicAndSpace(epicID, spaceID); err != nil {
		return domain.Todo{}, err
	}
	// 課題タイプや優先度を送らない呼び出し（ドラッグでの移動など）で値が変わらないよう、空なら今の値を使う
	todoType := domain.IssueType(issueType)
	todoPriority := domain.Priority(priority)
	if issueType == "" || priority == "" {
		current, err := s.repo.FindByID(id)
		if err != nil {
			return domain.Todo{}, err
		}
		if issueType == "" {
			todoType = current.IssueType
		}
		if priority == "" {
			todoPriority = current.Priority
		}
	}
	if !domain.ValidIssueType(todoType) {
		return domain.Todo{}, ErrInvalidIssueType
	}
	if !domain.ValidPriority(todoPriority) {
		return domain.Todo{}, ErrInvalidPriority
	}

	todo := domain.Todo{
		Title:       title,
		Done:        done,
		Deadline:    deadline,
		Status:      domain.Status(status),
		IssueType:   todoType,
		Priority:    todoPriority,
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
