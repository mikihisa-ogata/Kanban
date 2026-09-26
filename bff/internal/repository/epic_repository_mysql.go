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
	rows, err := r.db.Query(`SELECT id, title, space_id FROM epics ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	epics := []domain.Epic{}
	for rows.Next() {
		var epic domain.Epic
		if err := rows.Scan(&epic.ID, &epic.Title, &epic.SpaceID); err != nil {
			return nil, err
		}
		epics = append(epics, epic)
	}
	return epics, rows.Err()
}

func (r *epicMySQLRepository) Create(epic domain.Epic) error {
	_, err := r.db.Exec(`INSERT INTO epics (title, space_id) VALUES (?, ?)`, epic.Title, epic.SpaceID)
	return err
}

func (r *epicMySQLRepository) Update(id int, epic domain.Epic) error {
	result, err := r.db.Exec(`UPDATE epics SET title = ?, space_id = ? WHERE id = ?`, epic.Title, epic.SpaceID, id)
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
