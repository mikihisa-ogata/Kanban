package repository

import "todo-api/internal/domain"

type FilterRepository interface {
	// FindByOwner はユーザーの保存済みフィルターを名前順に返す
	FindByOwner(ownerID int) ([]domain.SavedFilter, error)
	// FindByID は保存済みフィルターを返す。存在しなければ nil を返す
	FindByID(id int) (*domain.SavedFilter, error)
	// Create は保存済みフィルターを作成し、その ID を返す。作成日時・更新日時は現在日時にする
	Create(filter domain.SavedFilter) (int, error)
	// Update は保存済みフィルターの名前・JQL・お気に入りを変え、更新日時を現在日時にする。存在しなければ not found のエラーを返す
	Update(filter domain.SavedFilter) error
	// Delete は保存済みフィルターを削除する。存在しなければ not found のエラーを返す
	Delete(id int) error
}

type IssueViewRepository interface {
	// Record はユーザーが課題を表示した日時を現在日時にする（課題 1 件につき最後の 1 回だけを残す）
	Record(userID int, todoID int) error
}
