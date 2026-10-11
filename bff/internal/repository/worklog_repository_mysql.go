package repository

import (
	"database/sql"
	"errors"
	"time"

	"todo-api/internal/domain"
)

type worklogMySQLRepository struct {
	db *sql.DB
}

func NewWorklogMySQLRepository(db *sql.DB) WorklogRepository {
	return &worklogMySQLRepository{db: db}
}

const selectWorklogs = `SELECT id, todo_id, minutes, started_at, note, created_at, updated_at FROM worklogs`

func (r *worklogMySQLRepository) FindByTodo(todoID int) ([]domain.Worklog, error) {
	rows, err := r.db.Query(selectWorklogs+` WHERE todo_id = ? ORDER BY started_at, id`, todoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	worklogs := []domain.Worklog{}
	for rows.Next() {
		worklog, err := scanWorklog(rows)
		if err != nil {
			return nil, err
		}
		worklogs = append(worklogs, worklog)
	}
	return worklogs, rows.Err()
}

func (r *worklogMySQLRepository) FindByID(id int) (*domain.Worklog, error) {
	worklog, err := scanWorklog(r.db.QueryRow(selectWorklogs+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &worklog, nil
}

func scanWorklog(row interface{ Scan(dest ...any) error }) (domain.Worklog, error) {
	var w domain.Worklog
	var updatedAt sql.NullTime
	if err := row.Scan(&w.ID, &w.TodoID, &w.Minutes, &w.StartedAt, &w.Note, &w.CreatedAt, &updatedAt); err != nil {
		return domain.Worklog{}, err
	}
	if updatedAt.Valid {
		w.UpdatedAt = &updatedAt.Time
	}
	return w, nil
}

func (r *worklogMySQLRepository) Create(worklog domain.Worklog) (int, error) {
	result, err := r.db.Exec(
		`INSERT INTO worklogs (todo_id, minutes, started_at, note, created_at) VALUES (?, ?, ?, ?, ?)`,
		worklog.TodoID, worklog.Minutes, worklog.StartedAt.UTC().Truncate(time.Second), worklog.Note, currentTime())
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (r *worklogMySQLRepository) Update(id int, worklog domain.Worklog) error {
	result, err := r.db.Exec(
		`UPDATE worklogs SET minutes = ?, started_at = ?, note = ?, updated_at = ? WHERE id = ?`,
		worklog.Minutes, worklog.StartedAt.UTC().Truncate(time.Second), worklog.Note, currentTime(), id)
	if err != nil {
		return err
	}
	return requireAffected(result, "worklog", id)
}

func (r *worklogMySQLRepository) Delete(id int) error {
	result, err := r.db.Exec(`DELETE FROM worklogs WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(result, "worklog", id)
}
