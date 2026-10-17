package service

import (
	"errors"
	"fmt"
	"time"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

var (
	ErrSprintNotFound      = errors.New("指定されたスプリントが存在しません")
	ErrUnassignedSprint    = errors.New("スペース未割り当ての課題にはスプリントを作れません")
	ErrSprintAlreadyActive = errors.New("このスペースには開始中のスプリントがあります。完了してから開始してください")
	ErrSprintNotFuture     = errors.New("開始できるのは未開始のスプリントだけです")
	ErrSprintNotActive     = errors.New("完了できるのは開始中のスプリントだけです")
	ErrSprintClosed        = errors.New("完了したスプリントは変更できません")
	ErrSprintNotDeletable  = errors.New("削除できるのは未開始のスプリントだけです")
	ErrInvalidSprintMoveTo = errors.New("未完了の課題の移し先は、同じスペースの未開始のスプリントかバックログを選んでください")
	ErrSprintOtherSpace    = errors.New("課題と同じスペースのスプリントを選んでください")
	ErrSubtaskSprint       = errors.New("サブタスクのスプリントは親課題と同じです。親課題のスプリントを変えてください")
	ErrSprintTodoNotFound  = errors.New("指定された課題が存在しません")
)

type SprintService interface {
	// GetSprints はすべてのスプリントを、スペース・作成順に返す
	GetSprints() ([]domain.Sprint, error)
	// CreateSprint はスペースに未開始のスプリントを作る。name が空なら「<キー> スプリント <番号>」にする
	CreateSprint(spaceID int, name string) (domain.Sprint, error)
	// UpdateSprint はスプリントの名前・ゴール・期間を変える（完了したスプリントは変えられない。開始中なら期間は必須）
	UpdateSprint(id int, name, goal, startDate, endDate string) (domain.Sprint, error)
	// StartSprint は未開始のスプリントを、名前・ゴール・期間を決めて開始する。スペースごとに同時に 1 つまで
	StartSprint(id int, name, goal, startDate, endDate string) (domain.Sprint, error)
	// CompleteSprint は開始中のスプリントを完了する。スプリントの課題を記録し、
	// 完了していない課題は moveTo（同じスペースの未開始のスプリント。0 はバックログ）へ移す。移した課題のスプリントの変更は actorID の変更として履歴に記録する
	CompleteSprint(id int, moveTo int, actorID int) (domain.Sprint, error)
	// DeleteSprint は未開始のスプリントを削除する。その課題はバックログに戻る（履歴に記録する）
	DeleteSprint(id int, actorID int) error
	// SetTodoSprint は課題（とそのサブタスク）を sprintID のスプリント（0 はバックログ）に入れ、変わった課題を履歴に記録する
	SetTodoSprint(todoID int, sprintID int, actorID int) error
	// GetSprintIssues は完了したときに記録したスプリントの課題を返す
	GetSprintIssues(id int) ([]domain.SprintIssue, error)
	// GetBurndown はスプリントのバーンダウンを、変更履歴から再計算して返す。日付は loc のタイムゾーンで区切る
	GetBurndown(id int, loc *time.Location) (domain.Burndown, error)
}

type sprintService struct {
	repo      repository.SprintRepository
	todoRepo  repository.TodoRepository
	spaceRepo repository.SpaceRepository
	history   repository.HistoryRepository
}

func NewSprintService(r repository.SprintRepository, t repository.TodoRepository, s repository.SpaceRepository, h repository.HistoryRepository) SprintService {
	return &sprintService{repo: r, todoRepo: t, spaceRepo: s, history: h}
}

func (s *sprintService) GetSprints() ([]domain.Sprint, error) {
	return s.repo.FindAll()
}

func (s *sprintService) CreateSprint(spaceID int, name string) (domain.Sprint, error) {
	if spaceID == 0 {
		return domain.Sprint{}, ErrUnassignedSprint
	}
	spaces, err := s.spaceRepo.FindAll()
	if err != nil {
		return domain.Sprint{}, err
	}
	var space *domain.Space
	for i := range spaces {
		if spaces[i].ID == spaceID {
			space = &spaces[i]
		}
	}
	if space == nil {
		return domain.Sprint{}, ErrSpaceNotFound
	}
	if name == "" {
		// Jira と同じく「KAN スプリント 3」のように、スペースのスプリントの通し番号を付ける
		sprints, err := s.repo.FindAll()
		if err != nil {
			return domain.Sprint{}, err
		}
		count := 0
		for _, sprint := range sprints {
			if sprint.SpaceID == spaceID {
				count++
			}
		}
		name = fmt.Sprintf("%s スプリント %d", space.Key, count+1)
	}
	name, err = domain.NormalizeSprintName(name)
	if err != nil {
		return domain.Sprint{}, err
	}
	id, err := s.repo.Create(domain.Sprint{SpaceID: spaceID, Name: name})
	if err != nil {
		return domain.Sprint{}, err
	}
	return s.find(id)
}

func (s *sprintService) UpdateSprint(id int, name, goal, startDate, endDate string) (domain.Sprint, error) {
	sprint, err := s.find(id)
	if err != nil {
		return domain.Sprint{}, err
	}
	if sprint.State == domain.SprintStateClosed {
		return domain.Sprint{}, ErrSprintClosed
	}
	if err := applySprintFields(&sprint, name, goal, startDate, endDate, sprint.State == domain.SprintStateActive); err != nil {
		return domain.Sprint{}, err
	}
	if err := s.repo.Update(sprint); err != nil {
		return domain.Sprint{}, err
	}
	return s.find(id)
}

func (s *sprintService) StartSprint(id int, name, goal, startDate, endDate string) (domain.Sprint, error) {
	sprint, err := s.find(id)
	if err != nil {
		return domain.Sprint{}, err
	}
	if sprint.State != domain.SprintStateFuture {
		return domain.Sprint{}, ErrSprintNotFuture
	}
	active, err := s.activeSprint(sprint.SpaceID)
	if err != nil {
		return domain.Sprint{}, err
	}
	if active != nil {
		return domain.Sprint{}, ErrSprintAlreadyActive
	}
	if err := applySprintFields(&sprint, name, goal, startDate, endDate, true); err != nil {
		return domain.Sprint{}, err
	}
	sprint.State = domain.SprintStateActive
	if err := s.repo.Update(sprint); err != nil {
		return domain.Sprint{}, err
	}
	return s.find(id)
}

func (s *sprintService) CompleteSprint(id int, moveTo int, actorID int) (domain.Sprint, error) {
	sprint, err := s.find(id)
	if err != nil {
		return domain.Sprint{}, err
	}
	if sprint.State != domain.SprintStateActive {
		return domain.Sprint{}, ErrSprintNotActive
	}
	if moveTo != 0 {
		target, err := s.repo.FindByID(moveTo)
		if err != nil {
			return domain.Sprint{}, err
		}
		if target == nil || target.SpaceID != sprint.SpaceID || target.State != domain.SprintStateFuture {
			return domain.Sprint{}, ErrInvalidSprintMoveTo
		}
	}

	todos, err := s.todoRepo.FindAll()
	if err != nil {
		return domain.Sprint{}, err
	}
	// サブタスクは親課題と一緒に動くので、記録するのは親課題（サブタスクでない課題）だけ
	issues := []domain.SprintIssue{}
	moveIDs := []int{}
	for _, todo := range todos {
		if todo.SprintID != id || todo.ParentID != 0 {
			continue
		}
		issues = append(issues, domain.SprintIssue{SprintID: id, TodoID: todo.ID, StoryPoints: todo.StoryPoints, Completed: todo.Done})
		if !todo.Done {
			moveIDs = append(moveIDs, todo.ID)
		}
	}
	if err := s.repo.Complete(id, issues, moveIDs, moveTo); err != nil {
		return domain.Sprint{}, err
	}
	moved := map[int]bool{}
	for _, todoID := range moveIDs {
		moved[todoID] = true
	}
	if err := s.recordSprintChanges(todos, func(todo domain.Todo) bool { return moved[todo.ID] || moved[todo.ParentID] }, moveTo, actorID); err != nil {
		return domain.Sprint{}, err
	}
	return s.find(id)
}

func (s *sprintService) DeleteSprint(id int, actorID int) error {
	sprint, err := s.find(id)
	if err != nil {
		return err
	}
	if sprint.State != domain.SprintStateFuture {
		return ErrSprintNotDeletable
	}
	// 削除するとスプリントの名前が分からなくなるので、履歴に残す課題は削除の前に読んでおく
	todos, err := s.todoRepo.FindAll()
	if err != nil {
		return err
	}
	names, err := s.sprintNames()
	if err != nil {
		return err
	}
	if err := s.repo.Delete(id); err != nil {
		return err
	}
	return s.addSprintHistory(todos, func(todo domain.Todo) bool { return todo.SprintID == id }, 0, names, actorID)
}

func (s *sprintService) SetTodoSprint(todoID int, sprintID int, actorID int) error {
	todos, err := s.todoRepo.FindAll()
	if err != nil {
		return err
	}
	var todo *domain.Todo
	for i := range todos {
		if todos[i].ID == todoID {
			todo = &todos[i]
		}
	}
	if todo == nil {
		return ErrSprintTodoNotFound
	}
	if todo.ParentID != 0 {
		return ErrSubtaskSprint
	}
	if sprintID != 0 {
		sprint, err := s.find(sprintID)
		if err != nil {
			return err
		}
		if sprint.State == domain.SprintStateClosed {
			return ErrSprintClosed
		}
		if sprint.SpaceID != todo.SpaceID {
			return ErrSprintOtherSpace
		}
	}
	if err := s.repo.SetTodoSprint(todoID, sprintID); err != nil {
		return err
	}
	return s.recordSprintChanges(todos, func(todo domain.Todo) bool { return todo.ID == todoID || todo.ParentID == todoID }, sprintID, actorID)
}

// recordSprintChanges は、todos（変更前の課題）のうち target に当たる課題のスプリントが sprintID に変わったことを履歴に記録する
func (s *sprintService) recordSprintChanges(todos []domain.Todo, target func(domain.Todo) bool, sprintID int, actorID int) error {
	names, err := s.sprintNames()
	if err != nil {
		return err
	}
	return s.addSprintHistory(todos, target, sprintID, names, actorID)
}

func (s *sprintService) addSprintHistory(todos []domain.Todo, target func(domain.Todo) bool, sprintID int, names map[int]string, actorID int) error {
	entries := []domain.HistoryEntry{}
	for _, todo := range todos {
		if !target(todo) {
			continue
		}
		if entry, ok := domain.SprintChange(todo.ID, todo.SprintID, sprintID, names); ok {
			entry.UserID = actorID
			entries = append(entries, entry)
		}
	}
	return s.history.Add(entries)
}

// sprintNames はスプリントの ID から名前への対応を返す
func (s *sprintService) sprintNames() (map[int]string, error) {
	sprints, err := s.repo.FindAll()
	if err != nil {
		return nil, err
	}
	names := map[int]string{}
	for _, sprint := range sprints {
		names[sprint.ID] = sprint.Name
	}
	return names, nil
}

// burndownFields はバーンダウンの再計算に使う履歴の項目
var burndownFields = []string{domain.HistoryFieldSprint, domain.HistoryFieldSpace, domain.HistoryFieldStatusCategory, domain.HistoryFieldDone, domain.HistoryFieldStoryPoints}

func (s *sprintService) GetBurndown(id int, loc *time.Location) (domain.Burndown, error) {
	sprint, err := s.find(id)
	if err != nil {
		return domain.Burndown{}, err
	}
	if sprint.State == domain.SprintStateFuture {
		return domain.Burndown{}, domain.ErrSprintNotStarted
	}
	todos, err := s.todoRepo.FindAll()
	if err != nil {
		return domain.Burndown{}, err
	}
	recorded, err := s.repo.FindIssues(id)
	if err != nil {
		return domain.Burndown{}, err
	}
	history, err := s.history.FindByFields(burndownFields)
	if err != nil {
		return domain.Burndown{}, err
	}
	return domain.ComputeBurndown(domain.BurndownInput{
		Sprint: sprint, Todos: todos, Recorded: recorded, History: history, Now: time.Now().UTC(), Location: loc,
	})
}

func (s *sprintService) GetSprintIssues(id int) ([]domain.SprintIssue, error) {
	if _, err := s.find(id); err != nil {
		return nil, err
	}
	return s.repo.FindIssues(id)
}

func (s *sprintService) find(id int) (domain.Sprint, error) {
	sprint, err := s.repo.FindByID(id)
	if err != nil {
		return domain.Sprint{}, err
	}
	if sprint == nil {
		return domain.Sprint{}, ErrSprintNotFound
	}
	return *sprint, nil
}

// activeSprint はスペースの開始中のスプリントを返す。なければ nil
func (s *sprintService) activeSprint(spaceID int) (*domain.Sprint, error) {
	sprints, err := s.repo.FindAll()
	if err != nil {
		return nil, err
	}
	for i := range sprints {
		if sprints[i].SpaceID == spaceID && sprints[i].State == domain.SprintStateActive {
			return &sprints[i], nil
		}
	}
	return nil, nil
}

// applySprintFields は名前・ゴール・期間を検証して sprint に入れる。needDates なら期間は必須
func applySprintFields(sprint *domain.Sprint, name, goal, startDate, endDate string, needDates bool) error {
	name, err := domain.NormalizeSprintName(name)
	if err != nil {
		return err
	}
	goal, err = domain.NormalizeSprintGoal(goal)
	if err != nil {
		return err
	}
	if err := domain.ValidateSprintDates(startDate, endDate, needDates); err != nil {
		return err
	}
	sprint.Name, sprint.Goal, sprint.StartDate, sprint.EndDate = name, goal, startDate, endDate
	return nil
}
