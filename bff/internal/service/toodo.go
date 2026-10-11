package service

import (
	"errors"
	"fmt"
	"sort"
	"strings"

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

var ErrInvalidLinkRelation = errors.New("リンクの種類は blocks・is blocked by・relates to・duplicates・is duplicated by・clones・is cloned by のいずれかにしてください")

var ErrLinkTargetNotFound = errors.New("リンク先の課題が存在しません")

var ErrSelfLink = errors.New("課題を自分自身にはリンクできません")

var ErrLinkDuplicate = errors.New("同じリンクが既にあります")

var ErrLinkNotFound = errors.New("課題リンクが見つかりません")

// DurationUpdate は更新での時間（初期見積もり・残り時間）の指定。
// Set が false なら今の値のまま、Value が空文字なら未設定にし、それ以外は「1w 2d 3h 30m」形式として読む
type DurationUpdate struct {
	Set   bool
	Value string
}

// EstimatesUpdate は更新での初期見積もりと残り時間の指定
type EstimatesUpdate struct {
	Original  DurationUpdate
	Remaining DurationUpdate
}

// parse は指定された時間を分にする。空文字は nil（未設定）
func (u DurationUpdate) parse() (*int, error) {
	if strings.TrimSpace(u.Value) == "" {
		return nil, nil
	}
	minutes, err := domain.ParseDuration(u.Value)
	if err != nil {
		return nil, err
	}
	return &minutes, nil
}

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
	CreateTodo(title string, deadline string, status StatusInput, issueType string, priority string, labels []string, storyPoints *float64, parentID int, epicID int, description string, spaceID int) (domain.Todo, error)
	// UpdateTodo はタスクを更新し、更新後のタスクを返す（スペースを変えると課題キーが変わる）。
	// 課題タイプ・優先度が空、ラベルが nil、ストーリーポイントの Set が false なら今の値のままにする。
	// サブタスクのスペースとエピックは親と同じにする。親のスペースやエピックを変えると、サブタスクも同じものに移す。
	// 残り時間が未設定のまま初期見積もりを設定すると、残り時間は「初期見積もり − 記録済みの作業時間」にする
	UpdateTodo(id int, title string, done bool, deadline string, status StatusInput, issueType string, priority string, labels []string, storyPoints StoryPointsUpdate, estimates EstimatesUpdate, epicID int, description string, spaceID int) (domain.Todo, error)
	// DeleteTodo はタスクを削除する。サブタスクがあれば一緒に削除し、それらの課題リンク・添付ファイルも削除する
	DeleteTodo(id int) error
	MoveTodo(id int, beforeID int) error
	// AddLink は id の課題から targetID の課題へのリンクを作り、リンクを含む id の課題を返す。
	// relation は id の課題から見た名前（例: "is blocked by"）。id の課題がなければ ErrIssueNotFound を返す
	AddLink(id int, relation string, targetID int) (domain.Todo, error)
	// DeleteLink は課題リンクを削除する。存在しなければ ErrLinkNotFound を返す
	DeleteLink(linkID int) error
	// GetHistory は課題の変更履歴を記録順（古い順）に返す。課題がなければ ErrIssueNotFound を返す
	GetHistory(id int) ([]domain.HistoryEntry, error)
}

type todoService struct {
	repo           repository.TodoRepository
	epicRepo       repository.EpicRepository
	spaceRepo      repository.SpaceRepository
	linkRepo       repository.IssueLinkRepository
	attachmentRepo repository.AttachmentRepository
	history        historyRecorder
	statuses       statusResolver
}

func NewTodoService(r repository.TodoRepository, e repository.EpicRepository, s repository.SpaceRepository, l repository.IssueLinkRepository, h repository.HistoryRepository, a repository.AttachmentRepository, st repository.StatusRepository) TodoService {
	return &todoService{repo: r, epicRepo: e, spaceRepo: s, linkRepo: l, attachmentRepo: a,
		history: historyRecorder{history: h, epics: e, spaces: s}, statuses: statusResolver{repo: st}}
}

func (s *todoService) GetHistory(id int) ([]domain.HistoryEntry, error) {
	todos, err := s.repo.FindAll()
	if err != nil {
		return nil, err
	}
	for _, todo := range todos {
		if todo.ID == id {
			return s.history.history.FindByTodo(id)
		}
	}
	return nil, ErrIssueNotFound
}

func (s *todoService) GetTodos() ([]domain.Todo, error) {
	todos, err := s.repo.FindAll()
	if err != nil {
		return nil, err
	}
	if err := s.attachLinks(todos, todos); err != nil {
		return nil, err
	}
	return todos, nil
}

// findWithLinks は課題リンクを含めて id の課題を返す
func (s *todoService) findWithLinks(id int) (domain.Todo, error) {
	todo, err := s.repo.FindByID(id)
	if err != nil {
		return domain.Todo{}, err
	}
	return s.withLinks(todo)
}

// withLinks は課題に課題リンクを読み込んで返す
func (s *todoService) withLinks(todo domain.Todo) (domain.Todo, error) {
	all, err := s.repo.FindAll()
	if err != nil {
		return domain.Todo{}, err
	}
	targets := []domain.Todo{todo}
	if err := s.attachLinks(targets, all); err != nil {
		return domain.Todo{}, err
	}
	return targets[0], nil
}

// attachLinks は targets の各課題に、その課題から見た課題リンクを設定する。リンク相手の概要は all から引く。
// 並びはリンクの種類の順（blocks, is blocked by, relates to, ...）で、同じ種類は作成順
func (s *todoService) attachLinks(targets []domain.Todo, all []domain.Todo) error {
	links, err := s.linkRepo.FindAll()
	if err != nil {
		return err
	}
	byID := map[int]domain.Todo{}
	for _, todo := range all {
		byID[todo.ID] = todo
	}
	for i := range targets {
		id := targets[i].ID
		issueLinks := []domain.IssueLink{}
		for _, link := range links {
			if link.SourceID != id && link.TargetID != id {
				continue
			}
			otherID := link.TargetID
			if link.SourceID != id {
				otherID = link.SourceID
			}
			other, ok := byID[otherID]
			if !ok {
				continue
			}
			issueLinks = append(issueLinks, domain.IssueLink{
				ID:       link.ID,
				Relation: link.RelationFrom(id),
				Issue: domain.LinkedIssue{
					ID:             other.ID,
					Key:            other.Key,
					Title:          other.Title,
					Status:         other.Status,
					StatusName:     other.StatusName,
					StatusCategory: other.StatusCategory,
					IssueType:      other.IssueType,
					Priority:       other.Priority,
				},
			})
		}
		sort.SliceStable(issueLinks, func(a, b int) bool {
			return domain.RelationOrder(issueLinks[a].Relation) < domain.RelationOrder(issueLinks[b].Relation)
		})
		targets[i].Links = issueLinks
	}
	return nil
}

func (s *todoService) AddLink(id int, relation string, targetID int) (domain.Todo, error) {
	linkType, outward, ok := domain.ParseRelation(relation)
	if !ok {
		return domain.Todo{}, ErrInvalidLinkRelation
	}
	if id == targetID {
		return domain.Todo{}, ErrSelfLink
	}
	todos, err := s.repo.FindAll()
	if err != nil {
		return domain.Todo{}, err
	}
	var sourceFound, targetFound bool
	for _, todo := range todos {
		sourceFound = sourceFound || todo.ID == id
		targetFound = targetFound || todo.ID == targetID
	}
	if !sourceFound {
		return domain.Todo{}, ErrIssueNotFound
	}
	if !targetFound {
		return domain.Todo{}, ErrLinkTargetNotFound
	}

	link := domain.Link{Type: linkType, SourceID: id, TargetID: targetID}
	if !outward {
		link.SourceID, link.TargetID = targetID, id
	}
	links, err := s.linkRepo.FindAll()
	if err != nil {
		return domain.Todo{}, err
	}
	for _, existing := range links {
		if existing.SameAs(link) {
			return domain.Todo{}, ErrLinkDuplicate
		}
	}
	if _, err := s.linkRepo.Create(link); err != nil {
		return domain.Todo{}, err
	}
	return s.findWithLinks(id)
}

func (s *todoService) DeleteLink(linkID int) error {
	links, err := s.linkRepo.FindAll()
	if err != nil {
		return err
	}
	for _, link := range links {
		if link.ID == linkID {
			return s.linkRepo.Delete(linkID)
		}
	}
	return ErrLinkNotFound
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
	return s.withLinks(*todo)
}

func (s *todoService) CreateTodo(title string, deadline string, status StatusInput, issueType string, priority string, labels []string, storyPoints *float64, parentID int, epicID int, description string, spaceID int) (domain.Todo, error) {
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

	// ステータスの指定がなければ、スペースのワークフローのカテゴリ To Do の先頭にする
	workflowStatus, err := s.statuses.resolve(spaceID, status, nil)
	if err != nil {
		return domain.Todo{}, err
	}

	todo := domain.Todo{
		Title:       title,
		Deadline:    deadline,
		IssueType:   todoType,
		Priority:    todoPriority,
		Labels:      todoLabels,
		StoryPoints: storyPoints,
		ParentID:    parentID,
		EpicID:      epicID,
		Description: description,
		SpaceID:     spaceID,
	}
	applyStatus(&todo, workflowStatus)
	id, err := s.repo.Create(todo)
	if err != nil {
		return domain.Todo{}, err
	}
	if err := s.history.recordCreated(id); err != nil {
		return domain.Todo{}, err
	}
	return s.findWithLinks(id)
}

func (s *todoService) UpdateTodo(id int, title string, done bool, deadline string, status StatusInput, issueType string, priority string, labels []string, storyPoints StoryPointsUpdate, estimates EstimatesUpdate, epicID int, description string, spaceID int) (domain.Todo, error) {
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
	original, remaining := current.OriginalEstimate, current.RemainingEstimate
	if estimates.Original.Set {
		if original, err = estimates.Original.parse(); err != nil {
			return domain.Todo{}, err
		}
	}
	if estimates.Remaining.Set {
		if remaining, err = estimates.Remaining.parse(); err != nil {
			return domain.Todo{}, err
		}
	} else if estimates.Original.Set && remaining == nil && original != nil {
		left := max(*original-current.TimeSpent, 0)
		remaining = &left
	}

	// ステータスはスペースのワークフローから決める。完了（Done）はステータスのカテゴリで決まり、done の指定は使わない
	workflowStatus, err := s.statuses.resolve(spaceID, status, &current)
	if err != nil {
		return domain.Todo{}, err
	}

	names, err := s.history.names()
	if err != nil {
		return domain.Todo{}, err
	}

	todo := domain.Todo{
		Title:       title,
		Deadline:    deadline,
		IssueType:   todoType,
		Priority:    todoPriority,
		Labels:      todoLabels,
		StoryPoints: todoPoints,
		EpicID:      epicID,
		Description: description,
		SpaceID:     spaceID,
		// 見積もりは指定がなければ今の値のまま
		OriginalEstimate:  original,
		RemainingEstimate: remaining,
	}
	applyStatus(&todo, workflowStatus)
	if err := s.repo.Update(id, todo); err != nil {
		return domain.Todo{}, err
	}
	updated, err := s.findWithLinks(id)
	if err != nil {
		return domain.Todo{}, err
	}
	if err := s.history.recordChanges(names, current, updated); err != nil {
		return domain.Todo{}, err
	}
	if current.SpaceID != spaceID || current.EpicID != epicID {
		if err := s.moveSubtasks(names, id, epicID, spaceID); err != nil {
			return domain.Todo{}, err
		}
	}
	return updated, nil
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

// moveSubtasks は親課題のサブタスクを、親と同じスペース・エピックに移す（スペースが変わると課題キーも変わる）。
// 移したサブタスクの変更も履歴に記録する
func (s *todoService) moveSubtasks(names domain.HistoryNames, parentID int, epicID int, spaceID int) error {
	todos, err := s.repo.FindAll()
	if err != nil {
		return err
	}
	for _, todo := range todos {
		if todo.ParentID != parentID {
			continue
		}
		moved := todo
		moved.EpicID, moved.SpaceID = epicID, spaceID
		if err := updateWithHistory(s.repo, s.history, s.statuses, names, todo, moved); err != nil {
			return fmt.Errorf("failed to move subtask %d: %w", todo.ID, err)
		}
	}
	return nil
}

// DeleteTodo は課題とサブタスクの添付ファイル（ファイル本体も）を消してから、課題を削除する
func (s *todoService) DeleteTodo(id int) error {
	todos, err := s.repo.FindAll()
	if err != nil {
		return err
	}
	for _, todo := range todos {
		if todo.ID != id && todo.ParentID != id {
			continue
		}
		attachments, err := s.attachmentRepo.FindByTodo(todo.ID)
		if err != nil {
			return err
		}
		for _, attachment := range attachments {
			if err := s.attachmentRepo.Delete(attachment.ID); err != nil {
				return fmt.Errorf("failed to delete attachment %d: %w", attachment.ID, err)
			}
		}
	}
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
