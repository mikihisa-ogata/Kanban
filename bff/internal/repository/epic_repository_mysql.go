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
	rows, err := r.db.Query(`SELECT id, title, space_id, created_at, updated_at FROM epics ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	epics := []domain.Epic{}
	for rows.Next() {
		var epic domain.Epic
		if err := rows.Scan(&epic.ID, &epic.Title, &epic.SpaceID, &epic.CreatedAt, &epic.UpdatedAt); err != nil {
			return nil, err
		}
		epics = append(epics, epic)
	}
	return epics, rows.Err()
}

// Create はエピックを追加する。作成日時・更新日時は現在日時にする
func (r *epicMySQLRepository) Create(epic domain.Epic) error {
	now := currentTime()
	_, err := r.db.Exec(`INSERT INTO epics (title, space_id, created_at, updated_at) VALUES (?, ?, ?, ?)`,
		epic.Title, epic.SpaceID, now, now)
	return err
}

// Update はエピックを更新し、更新日時を現在日時にする（作成日時は変えない）
func (r *epicMySQLRepository) Update(id int, epic domain.Epic) error {
	result, err := r.db.Exec(`UPDATE epics SET title = ?, space_id = ?, updated_at = ? WHERE id = ?`,
		epic.Title, epic.SpaceID, currentTime(), id)
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
