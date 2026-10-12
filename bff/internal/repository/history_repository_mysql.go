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
		`SELECT h.id, h.todo_id, h.field, h.old_value, h.new_value, h.created_at, h.user_id, u.display_name, u.avatar_color
		 FROM issue_history h LEFT JOIN users u ON u.id = h.user_id WHERE h.todo_id = ? ORDER BY h.id`, todoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []domain.HistoryEntry{}
	for rows.Next() {
		var entry domain.HistoryEntry
		var oldValue, newValue sql.NullString
		var user nullUser
		if err := rows.Scan(&entry.ID, &entry.TodoID, &entry.Field, &oldValue, &newValue, &entry.CreatedAt,
			&entry.UserID, &user.name, &user.color); err != nil {
			return nil, err
		}
		entry.User = user.summary(entry.UserID)
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
			`INSERT INTO issue_history (todo_id, field, old_value, new_value, user_id, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
			entry.TodoID, entry.Field, entry.OldValue, entry.NewValue, entry.UserID, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}
