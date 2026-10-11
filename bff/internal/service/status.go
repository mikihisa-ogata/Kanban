package service

import (
	"errors"

	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

var ErrStatusNotFound = errors.New("指定されたステータスが存在しません")

var ErrInvalidStatus = errors.New("ステータスは Open・InProgress・Waiting・Closed のいずれかか、ステータスの ID で指定してください")

// StatusInput は作成・更新でのステータスの指定。ID（ワークフローのステータスの ID）を優先し、
// 0 なら導入前の status 文字列（Open など）で指定する。どちらもなければ指定なし
type StatusInput struct {
	ID     int
	Legacy string
}

// statusResolver は課題のステータスを、課題のスペースのワークフローから決める
type statusResolver struct {
	repo repository.StatusRepository
}

// statusesOf はスペースのステータスを並び順に返す。まだなければ既定の 4 ステータスを作る
func (r statusResolver) statusesOf(spaceID int) ([]domain.WorkflowStatus, error) {
	if err := r.repo.EnsureDefaults(spaceID); err != nil {
		return nil, err
	}
	return r.repo.FindBySpace(spaceID)
}

// find は ID のステータスを返す。なければ nil
func (r statusResolver) find(id int) (*domain.WorkflowStatus, error) {
	all, err := r.repo.FindAll()
	if err != nil {
		return nil, err
	}
	for _, s := range all {
		if s.ID == id {
			return &s, nil
		}
	}
	return nil, nil
}

// currentOf は課題の今のステータスを返す。対応付け前（StatusID が 0）なら導入前の status からカテゴリを決める。
// 新しく作る課題なら nil
func (r statusResolver) currentOf(todo *domain.Todo) (*domain.WorkflowStatus, error) {
	if todo == nil {
		return nil, nil
	}
	if todo.StatusID != 0 {
		s, err := r.find(todo.StatusID)
		if err != nil || s != nil {
			return s, err
		}
	}
	category, ok := domain.LegacyCategory(todo.Status)
	if !ok {
		category = domain.StatusCategoryToDo
	}
	return &domain.WorkflowStatus{Name: todo.StatusName, Category: category, LegacyKey: todo.Status}, nil
}

// resolve は課題を spaceID のスペースに置くときのステータスを決める。current は課題の今の状態（新しく作るなら nil）。
//   - input.ID がそのスペースのステータスならそれ。別のスペースのステータスなら、名前かカテゴリが同じものに対応付ける
//   - input.Legacy（Open など）なら、そのスペースの同じ導入前のステータス、なければ同じカテゴリの先頭。
//     ただし今のステータスの互換用の値と同じなら（画面が今の値を送り返しただけなら）今のステータスのまま
//   - 指定がなければ今のステータス。スペースが変わったら、名前かカテゴリが同じものに対応付ける
//   - 新しく作る課題で指定がなければ、カテゴリ To Do の先頭
func (r statusResolver) resolve(spaceID int, input StatusInput, current *domain.Todo) (domain.WorkflowStatus, error) {
	statuses, err := r.statusesOf(spaceID)
	if err != nil {
		return domain.WorkflowStatus{}, err
	}
	currentStatus, err := r.currentOf(current)
	if err != nil {
		return domain.WorkflowStatus{}, err
	}
	inSpace := func(id int) (domain.WorkflowStatus, bool) {
		for _, s := range statuses {
			if id != 0 && s.ID == id {
				return s, true
			}
		}
		return domain.WorkflowStatus{}, false
	}

	switch {
	case input.ID != 0:
		if s, ok := inSpace(input.ID); ok {
			return s, nil
		}
		other, err := r.find(input.ID)
		if err != nil {
			return domain.WorkflowStatus{}, err
		}
		if other == nil {
			return domain.WorkflowStatus{}, ErrStatusNotFound
		}
		return matchOrFail(statuses, *other)
	case input.Legacy != "":
		legacy := domain.Status(input.Legacy)
		if currentStatus != nil && currentStatus.LegacyStatus() == legacy {
			if s, ok := inSpace(currentStatus.ID); ok {
				return s, nil
			}
		}
		for _, s := range statuses {
			if s.LegacyKey == legacy {
				return s, nil
			}
		}
		category, ok := domain.LegacyCategory(legacy)
		if !ok {
			return domain.WorkflowStatus{}, ErrInvalidStatus
		}
		return firstOrFail(statuses, category)
	case currentStatus != nil:
		if s, ok := inSpace(currentStatus.ID); ok {
			return s, nil
		}
		return matchOrFail(statuses, *currentStatus)
	}
	return firstOrFail(statuses, domain.StatusCategoryToDo)
}

func matchOrFail(statuses []domain.WorkflowStatus, from domain.WorkflowStatus) (domain.WorkflowStatus, error) {
	if s, ok := domain.MatchStatus(statuses, from); ok {
		return s, nil
	}
	return domain.WorkflowStatus{}, ErrStatusNotFound
}

func firstOrFail(statuses []domain.WorkflowStatus, category domain.StatusCategory) (domain.WorkflowStatus, error) {
	if s, ok := domain.FirstInCategory(statuses, category); ok {
		return s, nil
	}
	return domain.WorkflowStatus{}, ErrStatusNotFound
}

// applyStatus は課題をステータスにする。互換用の Status と完了（Done カテゴリなら完了）も合わせる
func applyStatus(todo *domain.Todo, s domain.WorkflowStatus) {
	todo.StatusID = s.ID
	todo.StatusName = s.Name
	todo.StatusCategory = s.Category
	todo.Status = s.LegacyStatus()
	todo.Done = s.Category == domain.StatusCategoryDone
}
