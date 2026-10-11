package service

import (
	"todo-api/internal/domain"
	"todo-api/internal/repository"
)

// historyRecorder は課題の作成と変更を変更履歴に記録する。課題を変えるサービスはすべてこれを通して記録する
type historyRecorder struct {
	history repository.HistoryRepository
	epics   repository.EpicRepository
	spaces  repository.SpaceRepository
}

// names は今のエピックとスペースの名前を返す。エピックやスペースを削除する場合は、削除の前に読んでおく
func (r historyRecorder) names() (domain.HistoryNames, error) {
	names := domain.HistoryNames{Epics: map[int]string{}, Spaces: map[int]string{}}
	epics, err := r.epics.FindAll()
	if err != nil {
		return names, err
	}
	for _, epic := range epics {
		names.Epics[epic.ID] = epic.Title
	}
	spaces, err := r.spaces.FindAll()
	if err != nil {
		return names, err
	}
	for _, space := range spaces {
		names.Spaces[space.ID] = space.Title
	}
	return names, nil
}

// recordCreated は課題の作成を記録する
func (r historyRecorder) recordCreated(todoID int) error {
	return r.history.Add([]domain.HistoryEntry{{TodoID: todoID, Field: domain.HistoryFieldCreated}})
}

// recordChanges は before から after への変更を項目ごとに記録する。変更がなければ何もしない
func (r historyRecorder) recordChanges(names domain.HistoryNames, before, after domain.Todo) error {
	return r.history.Add(domain.DiffTodo(before, after, names))
}

// updateWithHistory は before の課題を after の内容で更新し、変わった項目を履歴に記録する。
// エピックの移動やスペースの削除など、他の操作に伴って課題が変わるときに使う
func updateWithHistory(repo repository.TodoRepository, r historyRecorder, names domain.HistoryNames, before, after domain.Todo) error {
	if err := repo.Update(before.ID, after); err != nil {
		return err
	}
	updated, err := repo.FindByID(before.ID)
	if err != nil {
		return err
	}
	return r.recordChanges(names, before, updated)
}
