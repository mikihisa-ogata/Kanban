package repository

import (
	"database/sql"

	"todo-api/internal/domain"
)

type epicMySQLRepository struct {
	db *sql.DB
}

func NewEpicMySQLRepository(db *sql.DB) EpicRepository {
	return &epicMySQLRepository{db: db}
}

func (r *epicMySQLRepository) FindAll() ([]domain.Epic, error) {
	rows, err := r.db.Query(`SELECT id, title, space_id, start_date, due_date, color, created_at, updated_at FROM epics ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	epics := []domain.Epic{}
	for rows.Next() {
		var epic domain.Epic
		var start, due sql.NullTime
		if err := rows.Scan(&epic.ID, &epic.Title, &epic.SpaceID, &start, &due, &epic.Color, &epic.CreatedAt, &epic.UpdatedAt); err != nil {
			return nil, err
		}
		if start.Valid {
			epic.StartDate = start.Time.Format(deadlineLayout)
		}
		if due.Valid {
			epic.DueDate = due.Time.Format(deadlineLayout)
		}
		epics = append(epics, epic)
	}
	return epics, rows.Err()
}

// Create はエピックを追加する。作成日時・更新日時は現在日時にする
func (r *epicMySQLRepository) Create(epic domain.Epic) error {
	start, due, err := epicDates(epic)
	if err != nil {
		return err
	}
	now := currentTime()
	_, err = r.db.Exec(`INSERT INTO epics (title, space_id, start_date, due_date, color, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		epic.Title, epic.SpaceID, start, due, epic.Color, now, now)
	return err
}

// Update はエピックを更新し、更新日時を現在日時にする（作成日時は変えない）
func (r *epicMySQLRepository) Update(id int, epic domain.Epic) error {
	start, due, err := epicDates(epic)
	if err != nil {
		return err
	}
	result, err := r.db.Exec(`UPDATE epics SET title = ?, space_id = ?, start_date = ?, due_date = ?, color = ?, updated_at = ? WHERE id = ?`,
		epic.Title, epic.SpaceID, start, due, epic.Color, currentTime(), id)
	if err != nil {
		return err
	}
	return requireAffected(result, "epic", id)
}

func (r *epicMySQLRepository) Delete(id int) error {
	result, err := r.db.Exec(`DELETE FROM epics WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(result, "epic", id)
}

// epicDates は開始日・期限日を DB に保存する値にする（空文字は NULL）
func epicDates(epic domain.Epic) (sql.NullTime, sql.NullTime, error) {
	start, err := toNullDeadline(epic.StartDate)
	if err != nil {
		return sql.NullTime{}, sql.NullTime{}, err
	}
	due, err := toNullDeadline(epic.DueDate)
	if err != nil {
		return sql.NullTime{}, sql.NullTime{}, err
	}
	return start, due, nil
}
