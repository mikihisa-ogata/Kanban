package service

import (
	"errors"
	"fmt"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

var ErrEpicSpaceMismatch = errors.New("エピックとタスクのスペースが一致しません")

var ErrIssueNotFound = errors.New("課題が見つかりません")

var ErrInvalidIssueType = errors.New("課題タイプは Task・Bug・Story のいずれかにしてください")

var ErrInvalidPriority = errors.New("優先度は Highest・High・Medium・Low・Lowest のいずれかにしてください")

var ErrInvalidStoryPoints = errors.New("ストーリーポイントは 0 以上 999999.99 以下で、小数は第 2 位までにしてください")

var ErrParentNotFound = errors.New("親課題が存在しません")

var ErrSubtaskParent = errors.New("サブタスクの下にはサブタスクを作れません")

var ErrSubtaskType = errors.New("サブタスクは親課題を指定して作成し、他の課題タイプとは相互に変更できません")

// StoryPointsUpdate は更新でのストーリーポイントの指定。Set が false なら今の値のまま、Value が nil なら未設定にする
type StoryPointsUpdate struct {
	Set   bool
	Value *float64
}

type TodoService interface {
	GetTodos() ([]domain.Todo, error)
	// GetLabels はいずれかのタスクに付いているラベルを名前順に返す（入力欄の補完候補）
	GetLabels() ([]string, error)
	// GetTodoByKey は課題キー（例: KAN-12）のタスクを返す。形式が正しくないか存在しなければ ErrIssueNotFound を返す
	GetTodoByKey(key string) (domain.Todo, error)
	// CreateTodo はタスクを作成し、採番した課題キーを含む作成後のタスクを返す。課題タイプが空なら Task、優先度が空なら Medium にする
	// ストーリーポイントが nil なら未設定にする。
	// parentID を指定するとその課題のサブタスクを作る（課題タイプは Subtask、スペースとエピックは親と同じ）
	CreateTodo(title string, deadline string, status string, issueType string, priority string, labels []string, storyPoints *float64, parentID int, epicID int, description string, spaceID int) (domain.Todo, error)
	// UpdateTodo はタスクを更新し、更新後のタスクを返す（スペースを変えると課題キーが変わる）。
	// 課題タイプ・優先度が空、ラベルが nil、ストーリーポイントの Set が false なら今の値のままにする。
	// サブタスクのスペースとエピックは親と同じにする。親のスペースやエピックを変えると、サブタスクも同じものに移す
	UpdateTodo(id int, title string, done bool, deadline string, status string, issueType string, priority string, labels []string, storyPoints StoryPointsUpdate, epicID int, description string, spaceID int) (domain.Todo, error)
	// DeleteTodo はタスクを削除する。サブタスクがあれば一緒に削除する
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

func (s *todoService) GetLabels() ([]string, error) {
	return s.repo.FindLabels()
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

func (s *todoService) CreateTodo(title string, deadline string, status string, issueType string, priority string, labels []string, storyPoints *float64, parentID int, epicID int, description string, spaceID int) (domain.Todo, error) {
	todoType := domain.IssueType(issueType)
	if parentID != 0 {
		// サブタスクは親と同じスペース・エピックに置く
		parent, err := s.findParent(parentID)
		if err != nil {
			return domain.Todo{}, err
		}
		if issueType != "" && todoType != domain.IssueTypeSubtask {
			return domain.Todo{}, ErrSubtaskType
		}
		todoType = domain.IssueTypeSubtask
		epicID, spaceID = parent.EpicID, parent.SpaceID
	} else if todoType == domain.IssueTypeSubtask {
		return domain.Todo{}, ErrSubtaskType
	}
	if err := s.validateEpicAndSpace(epicID, spaceID); err != nil {
		return domain.Todo{}, err
	}
	if issueType == "" && parentID == 0 {
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
	todoLabels, err := domain.NormalizeLabels(labels)
	if err != nil {
		return domain.Todo{}, err
	}
	if storyPoints != nil && !domain.ValidStoryPoints(*storyPoints) {
		return domain.Todo{}, ErrInvalidStoryPoints
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
		Labels:      todoLabels,
		StoryPoints: storyPoints,
		ParentID:    parentID,
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

func (s *todoService) UpdateTodo(id int, title string, done bool, deadline string, status string, issueType string, priority string, labels []string, storyPoints StoryPointsUpdate, epicID int, description string, spaceID int) (domain.Todo, error) {
	current, err := s.repo.FindByID(id)
	if err != nil {
		return domain.Todo{}, err
	}
	// サブタスクは親と同じスペース・エピックに置く
	if current.ParentID != 0 {
		parent, err := s.findParent(current.ParentID)
		if err != nil {
			return domain.Todo{}, err
		}
		epicID, spaceID = parent.EpicID, parent.SpaceID
	}
	if err := s.validateEpicAndSpace(epicID, spaceID); err != nil {
		return domain.Todo{}, err
	}
	// 課題タイプ・優先度・ラベル・ストーリーポイントを送らない呼び出し（ドラッグでの移動など）で値が変わらないよう、
	// 送られなければ今の値を使う
	todoType := domain.IssueType(issueType)
	todoPriority := domain.Priority(priority)
	todoPoints := storyPoints.Value
	if issueType == "" {
		todoType = current.IssueType
	}
	if priority == "" {
		todoPriority = current.Priority
	}
	if labels == nil {
		labels = current.Labels
	}
	if !storyPoints.Set {
		todoPoints = current.StoryPoints
	}
	if !domain.ValidIssueType(todoType) {
		return domain.Todo{}, ErrInvalidIssueType
	}
	// サブタスクかどうかは作成時に決まり、課題タイプの変更では変えられない
	if (todoType == domain.IssueTypeSubtask) != (current.ParentID != 0) {
		return domain.Todo{}, ErrSubtaskType
	}
	if !domain.ValidPriority(todoPriority) {
		return domain.Todo{}, ErrInvalidPriority
	}
	todoLabels, err := domain.NormalizeLabels(labels)
	if err != nil {
		return domain.Todo{}, err
	}
	if todoPoints != nil && !domain.ValidStoryPoints(*todoPoints) {
		return domain.Todo{}, ErrInvalidStoryPoints
	}

	todo := domain.Todo{
		Title:       title,
		Done:        done,
		Deadline:    deadline,
		Status:      domain.Status(status),
		IssueType:   todoType,
		Priority:    todoPriority,
		Labels:      todoLabels,
		StoryPoints: todoPoints,
		EpicID:      epicID,
		Description: description,
		SpaceID:     spaceID,
	}
	if err := s.repo.Update(id, todo); err != nil {
		return domain.Todo{}, err
	}
	if current.SpaceID != spaceID || current.EpicID != epicID {
		if err := s.moveSubtasks(id, epicID, spaceID); err != nil {
			return domain.Todo{}, err
		}
	}
	return s.repo.FindByID(id)
}

// findParent は親課題にできる課題を返す。存在しなければ ErrParentNotFound、サブタスクなら ErrSubtaskParent を返す
func (s *todoService) findParent(id int) (domain.Todo, error) {
	todos, err := s.repo.FindAll()
	if err != nil {
		return domain.Todo{}, err
	}
	for _, todo := range todos {
		if todo.ID != id {
			continue
		}
		if todo.ParentID != 0 {
			return domain.Todo{}, ErrSubtaskParent
		}
		return todo, nil
	}
	return domain.Todo{}, ErrParentNotFound
}

// moveSubtasks は親課題のサブタスクを、親と同じスペース・エピックに移す（スペースが変わると課題キーも変わる）
func (s *todoService) moveSubtasks(parentID int, epicID int, spaceID int) error {
	todos, err := s.repo.FindAll()
	if err != nil {
		return err
	}
	for _, todo := range todos {
		if todo.ParentID != parentID {
			continue
		}
		todo.EpicID, todo.SpaceID = epicID, spaceID
		if err := s.repo.Update(todo.ID, todo); err != nil {
			return fmt.Errorf("failed to move subtask %d: %w", todo.ID, err)
		}
	}
	return nil
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
