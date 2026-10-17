package repository

import (
	"database/sql"
	"strings"
	"time"

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
		`SELECT h.id, h.todo_id, h.field, h.old_value, h.new_value, h.old_ref_id, h.new_ref_id, h.created_at, h.user_id, u.display_name, u.avatar_color
		 FROM issue_history h LEFT JOIN users u ON u.id = h.user_id WHERE h.todo_id = ? ORDER BY h.id`, todoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []domain.HistoryEntry{}
	for rows.Next() {
		var entry domain.HistoryEntry
		var values historyValues
		var user nullUser
		if err := rows.Scan(&entry.ID, &entry.TodoID, &entry.Field, &values.oldValue, &values.newValue, &values.oldRef, &values.newRef,
			&entry.CreatedAt, &entry.UserID, &user.name, &user.color); err != nil {
			return nil, err
		}
		entry.User = user.summary(entry.UserID)
		values.apply(&entry)
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
		if err := insertHistory(tx, entry, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *historyMySQLRepository) FindByFields(fields []string) ([]domain.HistoryEntry, error) {
	entries := []domain.HistoryEntry{}
	if len(fields) == 0 {
		return entries, nil
	}
	args := make([]any, len(fields))
	for i, field := range fields {
		args[i] = field
	}
	rows, err := r.db.Query(
		`SELECT id, todo_id, field, old_value, new_value, old_ref_id, new_ref_id, created_at, user_id
		 FROM issue_history WHERE field IN (?`+strings.Repeat(", ?", len(fields)-1)+`) ORDER BY id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var entry domain.HistoryEntry
		var values historyValues
		if err := rows.Scan(&entry.ID, &entry.TodoID, &entry.Field, &values.oldValue, &values.newValue, &values.oldRef, &values.newRef,
			&entry.CreatedAt, &entry.UserID); err != nil {
			return nil, err
		}
		values.apply(&entry)
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

// historyValues は履歴の変更前・変更後の値と ID の、NULL を含む読み出し先
type historyValues struct {
	oldValue, newValue sql.NullString
	oldRef, newRef     sql.NullInt64
}

func (v historyValues) apply(entry *domain.HistoryEntry) {
	if v.oldValue.Valid {
		entry.OldValue = &v.oldValue.String
	}
	if v.newValue.Valid {
		entry.NewValue = &v.newValue.String
	}
	if v.oldRef.Valid {
		id := int(v.oldRef.Int64)
		entry.OldRefID = &id
	}
	if v.newRef.Valid {
		id := int(v.newRef.Int64)
		entry.NewRefID = &id
	}
}

// insertHistory は変更履歴を 1 件記録する。日時は now、変更者は entry.UserID
func insertHistory(tx *sql.Tx, entry domain.HistoryEntry, now time.Time) error {
	_, err := tx.Exec(
		`INSERT INTO issue_history (todo_id, field, old_value, new_value, old_ref_id, new_ref_id, user_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		entry.TodoID, entry.Field, entry.OldValue, entry.NewValue, entry.OldRefID, entry.NewRefID, entry.UserID, now)
	return err
}
