package repository

import (
	"database/sql"

	"todo-api/internal/domain"
)

type historyMySQLRepository struct {
	db *sql.DB
}

func NewHistoryMySQLRepository(db *sql.DB) HistoryRepository {
	return &historyMySQLRepository{db: db}
}

func (r *historyMySQLRepository) FindByTodo(todoID int) ([]domain.HistoryEntry, error) {
	rows, err := r.db.Query(
		`SELECT id, todo_id, field, old_value, new_value, created_at FROM issue_history WHERE todo_id = ? ORDER BY id`, todoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []domain.HistoryEntry{}
	for rows.Next() {
		var entry domain.HistoryEntry
		var oldValue, newValue sql.NullString
		if err := rows.Scan(&entry.ID, &entry.TodoID, &entry.Field, &oldValue, &newValue, &entry.CreatedAt); err != nil {
			return nil, err
		}
		if oldValue.Valid {
			entry.OldValue = &oldValue.String
		}
		if newValue.Valid {
			entry.NewValue = &newValue.String
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// Add は変更履歴を 1 つのトランザクションで記録する（同じ変更の項目は同じ日時になる）
func (r *historyMySQLRepository) Add(entries []domain.HistoryEntry) error {
	if len(entries) == 0 {
		return nil
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := currentTime()
	for _, entry := range entries {
		if _, err := tx.Exec(
			`INSERT INTO issue_history (todo_id, field, old_value, new_value, created_at) VALUES (?, ?, ?, ?, ?)`,
			entry.TodoID, entry.Field, entry.OldValue, entry.NewValue, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}
