package service

import (
	"errors"
	"sort"
	"strings"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

var ErrBulkNoIssues = errors.New("課題を 1 件以上選んでください")

var ErrBulkNoChanges = errors.New("変更する項目を 1 つ以上指定してください")

// IDUpdate は一括変更でのスプリント・エピックの指定。Set が false なら今のまま、ID が 0 ならなし（スプリントはバックログ）にする
type IDUpdate struct {
	Set bool
	ID  int
}

// BulkUpdate は一括変更で変える項目。指定しなかった項目は今のまま
type BulkUpdate struct {
	// StatusID はワークフローのステータスの ID（0 は今のまま）。別のスペースの課題は、名前かカテゴリが同じステータスにする
	StatusID int
	Assignee UserUpdate
	// Priority は優先度（空文字は今のまま）
	Priority string
	// AddLabels・RemoveLabels は付けるラベルと外すラベル（外すほうは大文字・小文字を区別しない）
	AddLabels    []string
	RemoveLabels []string
	Sprint       IDUpdate
	Epic         IDUpdate
	// AddFixVersionIDs・RemoveFixVersionIDs は付ける修正バージョンと外す修正バージョン
	AddFixVersionIDs    []int
	RemoveFixVersionIDs []int
}

func (u BulkUpdate) empty() bool {
	return u.StatusID == 0 && !u.Assignee.Set && u.Priority == "" && len(u.AddLabels) == 0 && len(u.RemoveLabels) == 0 &&
		!u.Sprint.Set && !u.Epic.Set && len(u.AddFixVersionIDs) == 0 && len(u.RemoveFixVersionIDs) == 0
}

type BulkService interface {
	// UpdateIssues は ids の課題をまとめて変更し、内容が変わった課題の数を返す（Jira の一括変更）。
	// サブタスクのスプリント・エピックは親課題に合わせる（サブタスクに指定しても変えず、親課題を変えるとサブタスクも移す）。
	// 検証はすべて書き込む前に行い、書き込みは 1 つのトランザクションで全件の変更履歴と一緒に行う。
	// どれか 1 件でも変更できなければ、どの課題も変えずにエラーを返す。存在しない課題があれば ErrIssueNotFound
	UpdateIssues(ids []int, update BulkUpdate, actorID int) (int, error)
	// DeleteIssues は ids の課題（とそのサブタスク）を 1 つのトランザクションでまとめて削除し、削除した課題の数（サブタスクを含む）を返す。
	// 存在しない課題があれば、どれも削除せずに ErrIssueNotFound を返す
	DeleteIssues(ids []int) (int, error)
}

type bulkService struct {
	repo          repository.TodoRepository
	epics         repository.EpicRepository
	spaces        repository.SpaceRepository
	sprints       repository.SprintRepository
	versions      repository.VersionRepository
	users         repository.UserRepository
	attachments   repository.AttachmentRepository
	watchers      repository.WatcherRepository
	notifications repository.NotificationRepository
	statuses      statusResolver
}

func NewBulkService(r repository.TodoRepository, e repository.EpicRepository, s repository.SpaceRepository, sp repository.SprintRepository, v repository.VersionRepository, u repository.UserRepository, a repository.AttachmentRepository, st repository.StatusRepository, w repository.WatcherRepository, n repository.NotificationRepository) BulkService {
	return &bulkService{repo: r, epics: e, spaces: s, sprints: sp, versions: v, users: u, attachments: a, watchers: w, notifications: n, statuses: statusResolver{repo: st}}
}

// selectTodos は ids（重複は除く）の課題を指定順に返す。空なら ErrBulkNoIssues、存在しない課題があれば ErrIssueNotFound
func selectTodos(all []domain.Todo, ids []int) ([]domain.Todo, error) {
	byID := map[int]domain.Todo{}
	for _, todo := range all {
		byID[todo.ID] = todo
	}
	selected := []domain.Todo{}
	seen := map[int]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		todo, ok := byID[id]
		if !ok {
			return nil, ErrIssueNotFound
		}
		selected = append(selected, todo)
	}
	if len(selected) == 0 {
		return nil, ErrBulkNoIssues
	}
	return selected, nil
}

func (s *bulkService) UpdateIssues(ids []int, update BulkUpdate, actorID int) (int, error) {
	if update.empty() {
		return 0, ErrBulkNoChanges
	}
	all, err := s.repo.FindAll()
	if err != nil {
		return 0, err
	}
	selected, err := selectTodos(all, ids)
	if err != nil {
		return 0, err
	}
	plan, err := s.loadPlan(update)
	if err != nil {
		return 0, err
	}

	// 親課題を先に決め、そのあとでサブタスクのスプリント・エピックを親に合わせる
	after := map[int]domain.Todo{}
	order := []int{}
	for _, todo := range selected {
		changed, err := s.apply(todo, update, plan)
		if err != nil {
			return 0, err
		}
		after[todo.ID] = changed
		order = append(order, todo.ID)
	}
	if update.Sprint.Set || update.Epic.Set {
		for _, todo := range all {
			if todo.ParentID == 0 {
				continue
			}
			parent, ok := after[todo.ParentID]
			if !ok {
				continue
			}
			changed, ok := after[todo.ID]
			if !ok {
				changed = todo
				order = append(order, todo.ID)
			}
			changed.SprintID, changed.EpicID = parent.SprintID, parent.EpicID
			after[todo.ID] = changed
		}
	}

	before := map[int]domain.Todo{}
	for _, todo := range all {
		before[todo.ID] = todo
	}
	changes := []domain.BulkIssueChange{}
	for _, id := range order {
		old, changed := before[id], after[id]
		entries := append(domain.DiffTodo(old, changed, plan.names), domain.DiffPlanning(old, changed, plan.planning)...)
		if len(entries) == 0 {
			continue
		}
		for i := range entries {
			entries[i].UserID = actorID
		}
		change := domain.BulkIssueChange{Todo: changed, History: entries}
		// 新しい担当者は自動でウォッチャーになる（割り当てられたことも通知で届く）
		if changed.AssigneeID != old.AssigneeID {
			change.Watcher = changed.AssigneeID
		}
		changes = append(changes, change)
	}
	if len(changes) == 0 {
		return 0, nil
	}
	if err := s.repo.UpdateMany(changes); err != nil {
		return 0, err
	}
	// 変更を書き込んでから、課題ごとにウォッチャーへ「変更しました」の通知を作る
	for _, change := range changes {
		if err := notifyWatchers(s.watchers, s.notifications, change.Todo.ID, actorID, domain.NotificationUpdated, domain.ChangedFields(change.History), nil); err != nil {
			return 0, err
		}
	}
	return len(changes), nil
}

// bulkPlan は一括変更の検証と履歴に使う、指定された値と名前の対応
type bulkPlan struct {
	names    domain.HistoryNames
	planning domain.PlanningNames
	priority domain.Priority
	assignee *domain.UserSummary
	// addLabels は正規化した付けるラベル、removeLabels は外すラベル（小文字）
	addLabels    []string
	removeLabels map[string]bool
	epic         *domain.Epic
	sprint       *domain.Sprint
	versions     map[int]domain.Version
}

// loadPlan は指定された値を検証し、名前の対応と一緒に返す（課題ごとの検証は apply で行う）
func (s *bulkService) loadPlan(update BulkUpdate) (bulkPlan, error) {
	plan := bulkPlan{
		names:        domain.HistoryNames{Epics: map[int]string{}, Spaces: map[int]string{}},
		planning:     domain.PlanningNames{Sprints: map[int]string{}, Versions: map[int]string{}},
		removeLabels: map[string]bool{},
		versions:     map[int]domain.Version{},
	}
	epics, err := s.epics.FindAll()
	if err != nil {
		return plan, err
	}
	for i, epic := range epics {
		plan.names.Epics[epic.ID] = epic.Title
		if update.Epic.Set && epic.ID == update.Epic.ID {
			plan.epic = &epics[i]
		}
	}
	if update.Epic.Set && update.Epic.ID != 0 && plan.epic == nil {
		return plan, ErrEpicNotFound
	}
	spaces, err := s.spaces.FindAll()
	if err != nil {
		return plan, err
	}
	for _, space := range spaces {
		plan.names.Spaces[space.ID] = space.Title
	}
	sprints, err := s.sprints.FindAll()
	if err != nil {
		return plan, err
	}
	for i, sprint := range sprints {
		plan.planning.Sprints[sprint.ID] = sprint.Name
		if update.Sprint.Set && sprint.ID == update.Sprint.ID {
			plan.sprint = &sprints[i]
		}
	}
	if update.Sprint.Set && update.Sprint.ID != 0 {
		if plan.sprint == nil {
			return plan, ErrSprintNotFound
		}
		if plan.sprint.State == domain.SprintStateClosed {
			return plan, ErrSprintClosed
		}
	}
	versions, err := s.versions.FindAll()
	if err != nil {
		return plan, err
	}
	for _, version := range versions {
		plan.planning.Versions[version.ID] = version.Name
		plan.versions[version.ID] = version
	}
	for _, id := range append(append([]int{}, update.AddFixVersionIDs...), update.RemoveFixVersionIDs...) {
		if _, ok := plan.versions[id]; !ok {
			return plan, ErrVersionNotFound
		}
	}

	if update.Priority != "" {
		plan.priority = domain.Priority(update.Priority)
		if !domain.ValidPriority(plan.priority) {
			return plan, ErrInvalidPriority
		}
	}
	if update.Assignee.Set && update.Assignee.ID != 0 {
		user, err := s.users.FindByID(update.Assignee.ID)
		if err != nil {
			return plan, err
		}
		if user == nil {
			return plan, ErrUserNotFound
		}
		plan.assignee = &domain.UserSummary{ID: user.ID, DisplayName: user.DisplayName, AvatarColor: user.AvatarColor}
	}
	if plan.addLabels, err = domain.NormalizeLabels(update.AddLabels); err != nil {
		return plan, err
	}
	removeLabels, err := domain.NormalizeLabels(update.RemoveLabels)
	if err != nil {
		return plan, err
	}
	for _, label := range removeLabels {
		plan.removeLabels[strings.ToLower(label)] = true
	}
	return plan, nil
}

// apply は 1 件の課題に指定された変更を当てた結果を返す。この課題に当てられない変更があればエラーを返す
func (s *bulkService) apply(todo domain.Todo, update BulkUpdate, plan bulkPlan) (domain.Todo, error) {
	changed := todo
	if update.StatusID != 0 {
		status, err := s.statuses.resolve(todo.SpaceID, StatusInput{ID: update.StatusID}, &todo)
		if err != nil {
			return changed, err
		}
		applyStatus(&changed, status)
	}
	if update.Assignee.Set {
		changed.AssigneeID, changed.Assignee = update.Assignee.ID, plan.assignee
	}
	if plan.priority != "" {
		changed.Priority = plan.priority
	}
	if len(plan.addLabels) > 0 || len(plan.removeLabels) > 0 {
		labels := []string{}
		for _, label := range append(append([]string{}, todo.Labels...), plan.addLabels...) {
			if !plan.removeLabels[strings.ToLower(label)] {
				labels = append(labels, label)
			}
		}
		normalized, err := domain.NormalizeLabels(labels)
		if err != nil {
			return changed, err
		}
		changed.Labels = normalized
	}
	// サブタスクのスプリント・エピックは親課題に合わせるので、ここでは変えない
	if todo.ParentID == 0 {
		if update.Epic.Set {
			if plan.epic != nil && plan.epic.SpaceID != todo.SpaceID {
				return changed, ErrEpicSpaceMismatch
			}
			changed.EpicID = update.Epic.ID
		}
		if update.Sprint.Set {
			if plan.sprint != nil && plan.sprint.SpaceID != todo.SpaceID {
				return changed, ErrSprintOtherSpace
			}
			changed.SprintID = update.Sprint.ID
		}
	}
	if len(update.AddFixVersionIDs) > 0 || len(update.RemoveFixVersionIDs) > 0 {
		ids := []int{}
		for _, id := range todo.FixVersionIDs {
			if !containsInt(update.RemoveFixVersionIDs, id) {
				ids = append(ids, id)
			}
		}
		for _, id := range update.AddFixVersionIDs {
			if containsInt(ids, id) || containsInt(update.RemoveFixVersionIDs, id) {
				continue
			}
			version := plan.versions[id]
			if version.SpaceID != todo.SpaceID {
				return changed, ErrVersionOtherSpace
			}
			if version.Status == domain.VersionArchived {
				return changed, ErrVersionArchived
			}
			ids = append(ids, id)
		}
		sort.Ints(ids)
		changed.FixVersionIDs = ids
	}
	return changed, nil
}

func (s *bulkService) DeleteIssues(ids []int) (int, error) {
	all, err := s.repo.FindAll()
	if err != nil {
		return 0, err
	}
	selected, err := selectTodos(all, ids)
	if err != nil {
		return 0, err
	}
	// 親課題を消すとサブタスクも消えるので、親課題も選ばれているサブタスクは個別には消さない
	chosen := map[int]bool{}
	for _, todo := range selected {
		chosen[todo.ID] = true
	}
	deleteIDs := []int{}
	for _, todo := range selected {
		if todo.ParentID == 0 || !chosen[todo.ParentID] {
			deleteIDs = append(deleteIDs, todo.ID)
		}
	}
	removed := map[int]bool{}
	for _, todo := range all {
		if containsInt(deleteIDs, todo.ID) || containsInt(deleteIDs, todo.ParentID) {
			removed[todo.ID] = true
		}
	}
	files := []domain.Attachment{}
	for id := range removed {
		attachments, err := s.attachments.FindByTodo(id)
		if err != nil {
			return 0, err
		}
		files = append(files, attachments...)
	}
	if err := s.repo.DeleteMany(deleteIDs); err != nil {
		return 0, err
	}
	// 課題を消してから、添付ファイルの本体を消す（課題の削除が失敗してもファイルは残る）
	if err := s.attachments.RemoveFiles(files); err != nil {
		return 0, err
	}
	return len(removed), nil
}
