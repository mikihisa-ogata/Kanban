package repository

import (
	"database/sql"
	"fmt"
	"time"

	"todo-api/internal/domain"
)

// deadlineLayout は期限の形式（例: 2026-03-31）。空文字は期限なし（DB では NULL）
const deadlineLayout = "2006-01-02"

type todoMySQLRepository struct {
	db *sql.DB
}

func NewTodoMySQLRepository(db *sql.DB) TodoRepository {
	return &todoMySQLRepository{db: db}
}

// FindAll は表示順（position）でタスクを返す
func (r *todoMySQLRepository) FindAll() ([]domain.Todo, error) {
	rows, err := r.db.Query(
		`SELECT id, title, done, deadline, status, epic_id, description, space_id
		 FROM todos ORDER BY position, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	todos := []domain.Todo{}
	for rows.Next() {
		var todo domain.Todo
		var deadline sql.NullTime
		var status string
		if err := rows.Scan(&todo.ID, &todo.Title, &todo.Done, &deadline, &status,
			&todo.EpicID, &todo.Description, &todo.SpaceID); err != nil {
			return nil, err
		}
		todo.Status = domain.Status(status)
		if deadline.Valid {
			todo.Deadline = deadline.Time.Format(deadlineLayout)
		}
		todos = append(todos, todo)
	}
	return todos, rows.Err()
}

// Create はタスクを末尾に追加する
func (r *todoMySQLRepository) Create(todo domain.Todo) error {
	deadline, err := toNullDeadline(todo.Deadline)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(
		`INSERT INTO todos (title, done, deadline, status, epic_id, description, space_id, position)
		 SELECT ?, ?, ?, ?, ?, ?, ?, COALESCE(MAX(position), 0) + 1 FROM todos`,
		todo.Title, todo.Done, deadline, string(todo.Status), todo.EpicID, todo.Description, todo.SpaceID)
	return err
}

func (r *todoMySQLRepository) Update(id int, todo domain.Todo) error {
	deadline, err := toNullDeadline(todo.Deadline)
	if err != nil {
		return err
	}
	result, err := r.db.Exec(
		`UPDATE todos SET title = ?, done = ?, deadline = ?, status = ?, epic_id = ?, description = ?, space_id = ?
		 WHERE id = ?`,
		todo.Title, todo.Done, deadline, string(todo.Status), todo.EpicID, todo.Description, todo.SpaceID, id)
	if err != nil {
		return err
	}
	return requireAffected(result, "todo", id)
}

func (r *todoMySQLRepository) Delete(id int) error {
	result, err := r.db.Exec(`DELETE FROM todos WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(result, "todo", id)
}

// Move はタスクを beforeID のタスクの直前（0 なら末尾）へ移動し、全タスクの position を振り直す
func (r *todoMySQLRepository) Move(id int, beforeID int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	rows, err := tx.Query(`SELECT id FROM todos ORDER BY position, id FOR UPDATE`)
	if err != nil {
		return err
	}
	ids := []int{}
	for rows.Next() {
		var todoID int
		if err := rows.Scan(&todoID); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, todoID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	index := indexOf(ids, id)
	if index == -1 {
		return fmt.Errorf("todo with ID %d not found", id)
	}
	if beforeID == id {
		return nil
	}

	rest := append(ids[:index:index], ids[index+1:]...)
	insertAt := len(rest)
	if beforeID != 0 {
		insertAt = indexOf(rest, beforeID)
		if insertAt == -1 {
			return fmt.Errorf("todo with ID %d not found", beforeID)
		}
	}

	reordered := make([]int, 0, len(ids))
	reordered = append(reordered, rest[:insertAt]...)
	reordered = append(reordered, id)
	reordered = append(reordered, rest[insertAt:]...)

	for i, todoID := range reordered {
		if _, err := tx.Exec(`UPDATE todos SET position = ? WHERE id = ?`, i+1, todoID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// toNullDeadline は期限の文字列を DB の値に変換する。空文字は NULL にする
func toNullDeadline(deadline string) (sql.NullTime, error) {
	if deadline == "" {
		return sql.NullTime{}, nil
	}
	t, err := time.Parse(deadlineLayout, deadline)
	if err != nil {
		return sql.NullTime{}, fmt.Errorf("invalid deadline %q: %w", deadline, err)
	}
	return sql.NullTime{Time: t, Valid: true}, nil
}

// requireAffected は対象の行がなかった場合に not found のエラーを返す
func requireAffected(result sql.Result, name string, id int) error {
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("%s with ID %d not found", name, id)
	}
	return nil
}

func indexOf(ids []int, id int) int {
	for i, v := range ids {
		if v == id {
			return i
		}
	}
	return -1
}
