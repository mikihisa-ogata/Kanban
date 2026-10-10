package repository

import (
	"database/sql"
	"errors"
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

// selectTodos はタスクを読む SELECT。課題キーを作るためスペースのキーも読む（scanTodo と列を合わせる）
const selectTodos = `SELECT t.id, t.title, t.done, t.deadline, t.status, t.epic_id, t.description, t.space_id,
		        t.created_at, t.updated_at, t.number, s.space_key, t.issue_type, t.priority
		 FROM todos t LEFT JOIN spaces s ON s.id = t.space_id`

// FindAll は表示順（position）でタスクを返す。課題キーはスペースのキーと番号から作る
func (r *todoMySQLRepository) FindAll() ([]domain.Todo, error) {
	rows, err := r.db.Query(selectTodos + ` ORDER BY t.position, t.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	todos := []domain.Todo{}
	for rows.Next() {
		todo, err := scanTodo(rows)
		if err != nil {
			return nil, err
		}
		todos = append(todos, todo)
	}
	return todos, rows.Err()
}

// FindByID は指定 ID のタスクを返す。存在しなければ not found のエラーを返す
func (r *todoMySQLRepository) FindByID(id int) (domain.Todo, error) {
	todo, err := scanTodo(r.db.QueryRow(selectTodos+` WHERE t.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Todo{}, fmt.Errorf("todo with ID %d not found", id)
	}
	return todo, err
}

// FindByKey は課題キー（接頭辞と番号）のタスクを返す。domain.UnassignedKeyPrefix はスペース未割り当てのタスクを表す。
// 存在しなければ nil を返す
func (r *todoMySQLRepository) FindByKey(prefix string, number int) (*domain.Todo, error) {
	var row *sql.Row
	if prefix == domain.UnassignedKeyPrefix {
		row = r.db.QueryRow(selectTodos+` WHERE t.space_id = 0 AND t.number = ?`, number)
	} else {
		row = r.db.QueryRow(selectTodos+` WHERE s.space_key = ? AND t.number = ?`, prefix, number)
	}
	todo, err := scanTodo(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &todo, nil
}

// scanTodo は selectTodos の 1 行をタスクにする
func scanTodo(row interface{ Scan(dest ...any) error }) (domain.Todo, error) {
	var todo domain.Todo
	var deadline sql.NullTime
	var status string
	var spaceKey sql.NullString
	var issueType, priority string
	if err := row.Scan(&todo.ID, &todo.Title, &todo.Done, &deadline, &status,
		&todo.EpicID, &todo.Description, &todo.SpaceID, &todo.CreatedAt, &todo.UpdatedAt,
		&todo.Number, &spaceKey, &issueType, &priority); err != nil {
		return domain.Todo{}, err
	}
	todo.IssueType = domain.IssueType(issueType)
	todo.Priority = domain.Priority(priority)
	todo.Key = issueKey(todo.SpaceID, spaceKey.String, todo.Number)
	todo.Status = domain.Status(status)
	if deadline.Valid {
		todo.Deadline = deadline.Time.Format(deadlineLayout)
	}
	return todo, nil
}

// Create はタスクを末尾に追加し、スペース内の次の番号を振る。作成日時・更新日時は現在日時にする。
// 作成したタスクの ID を返す
func (r *todoMySQLRepository) Create(todo domain.Todo) (int, error) {
	deadline, err := toNullDeadline(todo.Deadline)
	if err != nil {
		return 0, err
	}
	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	number, err := nextIssueNumber(tx, todo.SpaceID)
	if err != nil {
		return 0, err
	}
	now := currentTime()
	result, err := tx.Exec(
		`INSERT INTO todos (title, done, deadline, status, issue_type, priority, epic_id, description, space_id, number, position, created_at, updated_at)
		 SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, COALESCE(MAX(position), 0) + 1, ?, ? FROM todos`,
		todo.Title, todo.Done, deadline, string(todo.Status), issueTypeOrDefault(todo.IssueType), priorityOrDefault(todo.Priority), todo.EpicID, todo.Description, todo.SpaceID, number, now, now)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return int(id), nil
}

// Update はタスクを更新し、更新日時を現在日時にする（作成日時は変えない）。
// スペースが変わった場合は、Jira で課題を別のプロジェクトへ移したときと同じく、移動先のスペースで番号を振り直す
func (r *todoMySQLRepository) Update(id int, todo domain.Todo) error {
	deadline, err := toNullDeadline(todo.Deadline)
	if err != nil {
		return err
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var spaceID, number int
	err = tx.QueryRow(`SELECT space_id, number FROM todos WHERE id = ? FOR UPDATE`, id).Scan(&spaceID, &number)
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("todo with ID %d not found", id)
	}
	if err != nil {
		return err
	}
	if spaceID != todo.SpaceID {
		if number, err = nextIssueNumber(tx, todo.SpaceID); err != nil {
			return err
		}
	}

	_, err = tx.Exec(
		`UPDATE todos SET title = ?, done = ?, deadline = ?, status = ?, issue_type = ?, priority = ?, epic_id = ?, description = ?, space_id = ?, number = ?, updated_at = ?
		 WHERE id = ?`,
		todo.Title, todo.Done, deadline, string(todo.Status), issueTypeOrDefault(todo.IssueType), priorityOrDefault(todo.Priority), todo.EpicID, todo.Description, todo.SpaceID, number, currentTime(), id)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *todoMySQLRepository) Delete(id int) error {
	result, err := r.db.Exec(`DELETE FROM todos WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(result, "todo", id)
}

// Move はタスクを beforeID のタスクの直前（0 なら末尾）へ移動し、全タスクの position を振り直す。
// 並び替えは内容の変更ではないため、更新日時は変えない
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

// nextIssueNumber はスペース内の次の番号を採番して返す。番号は issue_counters で管理し、タスクを削除しても再利用しない
func nextIssueNumber(tx *sql.Tx, spaceID int) (int, error) {
	if _, err := tx.Exec(
		`INSERT INTO issue_counters (space_id, last_number) VALUES (?, 1)
		 ON DUPLICATE KEY UPDATE last_number = last_number + 1`, spaceID); err != nil {
		return 0, err
	}
	var number int
	err := tx.QueryRow(`SELECT last_number FROM issue_counters WHERE space_id = ?`, spaceID).Scan(&number)
	return number, err
}

// issueKey は課題キーを返す。スペース未割り当てのタスクは domain.UnassignedKeyPrefix を接頭辞にする
func issueKey(spaceID int, spaceKey string, number int) string {
	if spaceID == 0 {
		return domain.IssueKey(domain.UnassignedKeyPrefix, number)
	}
	return domain.IssueKey(spaceKey, number)
}

// issueTypeOrDefault は保存する課題タイプを返す。空の場合は Task にする
func issueTypeOrDefault(t domain.IssueType) string {
	if t == "" {
		return string(domain.IssueTypeTask)
	}
	return string(t)
}

// priorityOrDefault は保存する優先度を返す。空の場合は Medium にする
func priorityOrDefault(p domain.Priority) string {
	if p == "" {
		return string(domain.PriorityMedium)
	}
	return string(p)
}

// currentTime は作成日時・更新日時に記録する現在日時を返す。DB の DATETIME に合わせて秒単位の UTC にする
func currentTime() time.Time {
	return time.Now().UTC().Truncate(time.Second)
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
