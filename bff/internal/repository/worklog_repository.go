package repository

import "todo-api/internal/domain"

type WorklogRepository interface {
	// FindByTodo は課題の作業ログを開始日時の順（古い順）に返す
	FindByTodo(todoID int) ([]domain.Worklog, error)
	// FindByID は作業ログを返す。存在しなければ nil を返す
	FindByID(id int) (*domain.Worklog, error)
	// Create は作業ログを作成し、その ID を返す。記録日時は現在日時にする
	Create(worklog domain.Worklog) (int, error)
	// Update は作業時間・開始日時・メモを変え、編集日時を現在日時にする。存在しなければ not found のエラーを返す
	Update(id int, worklog domain.Worklog) error
	// Delete は作業ログを削除する。存在しなければ not found のエラーを返す
	Delete(id int) error
}
