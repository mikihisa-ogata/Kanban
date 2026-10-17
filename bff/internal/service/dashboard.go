package service

import (
	"errors"
	"time"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

// ActivityLimitMax は最近のアクティビティで返せる件数の上限、ActivityLimitDefault は既定の件数
const (
	ActivityLimitDefault = 30
	ActivityLimitMax     = 100
)

var ErrInvalidActivityLimit = errors.New("件数は 1〜100 の整数で指定してください")

type DashboardService interface {
	// GetDashboard はユーザーのガジェットの並びを返す。保存していなければ既定の並び。
	// 消えたフィルターやほかのユーザーのフィルターを指すガジェットは、フィルターを未選択（0）にする
	GetDashboard(userID int) ([]domain.DashboardGadget, error)
	// SaveDashboard はユーザーのガジェットの並びを保存して、保存したものを返す。誤りは domain.ErrTooManyGadgets など
	SaveDashboard(userID int, gadgets []domain.DashboardGadget) ([]domain.DashboardGadget, error)
	// GetActivity はすべての課題の変更履歴を新しい順に limit 件（1〜ActivityLimitMax）まで、課題のキーと要約を付けて返す
	GetActivity(limit int) ([]domain.ActivityItem, error)
	// CountByStatus は JQL に合う課題をステータスごとに数える。JQL の誤りは *JQLError
	CountByStatus(jql string, actorID int) ([]domain.StatusCount, error)
	// CreatedVsResolved は JQL に合う課題の、今日までの days 日間の日ごとの作成数と解決数を返す
	CreatedVsResolved(jql string, days int, actorID int) (domain.CreatedVsResolved, error)
}

type dashboardService struct {
	repo        repository.DashboardRepository
	filterRepo  repository.FilterRepository
	historyRepo repository.HistoryRepository
	todoRepo    repository.TodoRepository
	jql         JQLService
	now         func() time.Time
	location    *time.Location
}

func NewDashboardService(r repository.DashboardRepository, f repository.FilterRepository, h repository.HistoryRepository, t repository.TodoRepository, j JQLService) DashboardService {
	return &dashboardService{repo: r, filterRepo: f, historyRepo: h, todoRepo: t, jql: j, now: time.Now, location: time.Local}
}

func (s *dashboardService) GetDashboard(userID int) ([]domain.DashboardGadget, error) {
	gadgets, err := s.repo.FindByUser(userID)
	if err != nil {
		return nil, err
	}
	if gadgets == nil {
		gadgets = domain.DefaultDashboard()
	}
	return s.dropForeignFilters(userID, gadgets)
}

func (s *dashboardService) SaveDashboard(userID int, gadgets []domain.DashboardGadget) ([]domain.DashboardGadget, error) {
	gadgets, err := domain.ValidateDashboard(gadgets)
	if err != nil {
		return nil, err
	}
	if gadgets, err = s.dropForeignFilters(userID, gadgets); err != nil {
		return nil, err
	}
	if err := s.repo.Save(userID, gadgets); err != nil {
		return nil, err
	}
	return gadgets, nil
}

// dropForeignFilters は userID のユーザーのものではないフィルター（消えたものを含む）を指すガジェットの FilterID を 0 にする
func (s *dashboardService) dropForeignFilters(userID int, gadgets []domain.DashboardGadget) ([]domain.DashboardGadget, error) {
	owned := map[int]bool{}
	filters, err := s.filterRepo.FindByOwner(userID)
	if err != nil {
		return nil, err
	}
	for _, f := range filters {
		owned[f.ID] = true
	}
	result := make([]domain.DashboardGadget, len(gadgets))
	for i, g := range gadgets {
		if g.FilterID != 0 && !owned[g.FilterID] {
			g.FilterID = 0
		}
		result[i] = g
	}
	return result, nil
}

func (s *dashboardService) GetActivity(limit int) ([]domain.ActivityItem, error) {
	if limit < 1 || limit > ActivityLimitMax {
		return nil, ErrInvalidActivityLimit
	}
	entries, err := s.historyRepo.FindRecent(limit)
	if err != nil {
		return nil, err
	}
	todos, err := s.todoRepo.FindAll()
	if err != nil {
		return nil, err
	}
	byID := map[int]domain.Todo{}
	for _, todo := range todos {
		byID[todo.ID] = todo
	}
	items := []domain.ActivityItem{}
	for _, entry := range entries {
		// 課題を消すと履歴も消えるが、念のため見つからない課題の履歴は出さない
		todo, ok := byID[entry.TodoID]
		if !ok {
			continue
		}
		items = append(items, domain.ActivityItem{HistoryEntry: entry, Key: todo.Key, Title: todo.Title})
	}
	return items, nil
}

func (s *dashboardService) CountByStatus(jql string, actorID int) ([]domain.StatusCount, error) {
	todos, err := s.matchingTodos(jql, actorID)
	if err != nil {
		return nil, err
	}
	return domain.CountByStatus(todos), nil
}

func (s *dashboardService) CreatedVsResolved(jql string, days int, actorID int) (domain.CreatedVsResolved, error) {
	if days < 1 || days > domain.CreatedVsResolvedDaysMax {
		return domain.CreatedVsResolved{}, domain.ErrInvalidTrendDays
	}
	todos, err := s.matchingTodos(jql, actorID)
	if err != nil {
		return domain.CreatedVsResolved{}, err
	}
	history, err := s.historyRepo.FindByFields([]string{domain.HistoryFieldStatusCategory, domain.HistoryFieldDone, domain.HistoryFieldStatus})
	if err != nil {
		return domain.CreatedVsResolved{}, err
	}
	return domain.ComputeCreatedVsResolved(todos, history, days, s.now(), s.location)
}

// matchingTodos は JQL に合う課題を返す
func (s *dashboardService) matchingTodos(jql string, actorID int) ([]domain.Todo, error) {
	ids, err := s.jql.MatchingIDs(jql, actorID)
	if err != nil {
		return nil, err
	}
	match := map[int]bool{}
	for _, id := range ids {
		match[id] = true
	}
	all, err := s.todoRepo.FindAll()
	if err != nil {
		return nil, err
	}
	todos := []domain.Todo{}
	for _, todo := range all {
		if match[todo.ID] {
			todos = append(todos, todo)
		}
	}
	return todos, nil
}
