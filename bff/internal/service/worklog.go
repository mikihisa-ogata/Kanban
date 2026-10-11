package service

import (
	"errors"
	"time"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

var ErrWorklogNotFound = errors.New("作業ログが見つかりません")

var ErrInvalidWorklogTime = errors.New("作業時間は 1 分以上を「1w 2d 3h 30m」の形式で入力してください")

var ErrInvalidWorklogStart = errors.New("開始日時を指定してください")

type WorklogService interface {
	// GetWorklogs は課題の作業ログを開始日時の順に返す。課題がなければ ErrIssueNotFound を返す
	GetWorklogs(todoID int) ([]domain.Worklog, error)
	// AddWorklog は課題に作業を記録し、作成した作業ログを返す。timeSpent は「1w 2d 3h 30m」形式。
	// remaining の Set が false なら残り時間を作業時間の分だけ自動で減らし（0 未満にはしない）、true なら指定の値にする
	AddWorklog(todoID int, timeSpent string, startedAt time.Time, note string, remaining DurationUpdate) (domain.Worklog, error)
	// UpdateWorklog は作業ログを変え、変更後の作業ログを返す。残り時間は作業時間の増減に合わせて自動で調整するか、remaining の値にする
	UpdateWorklog(id int, timeSpent string, startedAt time.Time, note string, remaining DurationUpdate) (domain.Worklog, error)
	// DeleteWorklog は作業ログを削除し、残り時間をその作業時間の分だけ戻す。なければ ErrWorklogNotFound を返す
	DeleteWorklog(id int) error
}

type worklogService struct {
	worklogRepo repository.WorklogRepository
	todoRepo    repository.TodoRepository
	history     historyRecorder
	statuses    statusResolver
}

func NewWorklogService(w repository.WorklogRepository, t repository.TodoRepository, e repository.EpicRepository, s repository.SpaceRepository, h repository.HistoryRepository, st repository.StatusRepository) WorklogService {
	return &worklogService{worklogRepo: w, todoRepo: t, history: historyRecorder{history: h, epics: e, spaces: s}, statuses: statusResolver{repo: st}}
}

func (s *worklogService) GetWorklogs(todoID int) ([]domain.Worklog, error) {
	if err := requireTodo(s.todoRepo, todoID); err != nil {
		return nil, err
	}
	return s.worklogRepo.FindByTodo(todoID)
}

func (s *worklogService) AddWorklog(todoID int, timeSpent string, startedAt time.Time, note string, remaining DurationUpdate) (domain.Worklog, error) {
	worklog, err := newWorklog(timeSpent, startedAt, note)
	if err != nil {
		return domain.Worklog{}, err
	}
	if err := requireTodo(s.todoRepo, todoID); err != nil {
		return domain.Worklog{}, err
	}
	worklog.TodoID = todoID
	id, err := s.worklogRepo.Create(worklog)
	if err != nil {
		return domain.Worklog{}, err
	}
	if err := s.adjustRemaining(todoID, worklog.Minutes, remaining); err != nil {
		return domain.Worklog{}, err
	}
	return s.findWorklog(id)
}

func (s *worklogService) UpdateWorklog(id int, timeSpent string, startedAt time.Time, note string, remaining DurationUpdate) (domain.Worklog, error) {
	worklog, err := newWorklog(timeSpent, startedAt, note)
	if err != nil {
		return domain.Worklog{}, err
	}
	current, err := s.findWorklog(id)
	if err != nil {
		return domain.Worklog{}, err
	}
	if err := s.worklogRepo.Update(id, worklog); err != nil {
		return domain.Worklog{}, err
	}
	if err := s.adjustRemaining(current.TodoID, worklog.Minutes-current.Minutes, remaining); err != nil {
		return domain.Worklog{}, err
	}
	return s.findWorklog(id)
}

func (s *worklogService) DeleteWorklog(id int) error {
	current, err := s.findWorklog(id)
	if err != nil {
		return err
	}
	if err := s.worklogRepo.Delete(id); err != nil {
		return err
	}
	return s.adjustRemaining(current.TodoID, -current.Minutes, DurationUpdate{})
}

// newWorklog は入力を検証して作業ログにする（課題はまだ決めない）
func newWorklog(timeSpent string, startedAt time.Time, note string) (domain.Worklog, error) {
	minutes, err := domain.ParseDuration(timeSpent)
	if err != nil || minutes <= 0 {
		return domain.Worklog{}, ErrInvalidWorklogTime
	}
	if startedAt.IsZero() {
		return domain.Worklog{}, ErrInvalidWorklogStart
	}
	if err := domain.ValidateWorklogNote(note); err != nil {
		return domain.Worklog{}, err
	}
	return domain.Worklog{Minutes: minutes, StartedAt: startedAt, Note: note}, nil
}

// adjustRemaining は作業時間が delta 分だけ増えた（負なら減った）課題の残り時間を調整し、変わったら履歴に記録する。
// remaining の Set が true ならその値にする。false なら残り時間を delta だけ減らす（0 未満にはしない）。
// 残り時間が未設定で初期見積もりがある場合は「初期見積もり − 記録済みの作業時間」にし、どちらも未設定なら変えない
func (s *worklogService) adjustRemaining(todoID int, delta int, remaining DurationUpdate) error {
	before, err := s.todoRepo.FindByID(todoID)
	if err != nil {
		return err
	}
	after := before
	if remaining.Set {
		if after.RemainingEstimate, err = remaining.parse(); err != nil {
			return err
		}
	} else if before.RemainingEstimate != nil {
		left := max(*before.RemainingEstimate-delta, 0)
		after.RemainingEstimate = &left
	} else if before.OriginalEstimate != nil {
		// before.TimeSpent は記録・変更・削除の後の合計
		left := max(*before.OriginalEstimate-before.TimeSpent, 0)
		after.RemainingEstimate = &left
	}
	if equalMinutes(before.RemainingEstimate, after.RemainingEstimate) {
		return nil
	}
	names, err := s.history.names()
	if err != nil {
		return err
	}
	return updateWithHistory(s.todoRepo, s.history, s.statuses, names, before, after)
}

func (s *worklogService) findWorklog(id int) (domain.Worklog, error) {
	worklog, err := s.worklogRepo.FindByID(id)
	if err != nil {
		return domain.Worklog{}, err
	}
	if worklog == nil {
		return domain.Worklog{}, ErrWorklogNotFound
	}
	return *worklog, nil
}

func equalMinutes(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
