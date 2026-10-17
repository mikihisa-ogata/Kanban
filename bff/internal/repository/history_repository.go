package repository

import "todo-api/internal/domain"

type HistoryRepository interface {
	// FindByTodo は課題の変更履歴を記録順（古い順）に返す
	FindByTodo(todoID int) ([]domain.HistoryEntry, error)
	// Add は変更履歴を記録する。変更者は各 entry の UserID、記録日時は現在日時にする
	Add(entries []domain.HistoryEntry) error
	// FindByFields は fields の項目の変更履歴を、すべての課題について記録順に返す（変更者の概要は読まない）
	FindByFields(fields []string) ([]domain.HistoryEntry, error)
}
