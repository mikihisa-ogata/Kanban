package repository

import (
	"database/sql"

	"todo-api/internal/domain"
)

type spaceMySQLRepository struct {
	db *sql.DB
}

func NewSpaceMySQLRepository(db *sql.DB) SpaceRepository {
	return &spaceMySQLRepository{db: db}
}

func (r *spaceMySQLRepository) FindAll() ([]domain.Space, error) {
	rows, err := r.db.Query(`SELECT id, title FROM spaces ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	spaces := []domain.Space{}
	for rows.Next() {
		var space domain.Space
		if err := rows.Scan(&space.ID, &space.Title); err != nil {
			return nil, err
		}
		spaces = append(spaces, space)
	}
	return spaces, rows.Err()
}

func (r *spaceMySQLRepository) Create(space domain.Space) error {
	_, err := r.db.Exec(`INSERT INTO spaces (title) VALUES (?)`, space.Title)
	return err
}

func (r *spaceMySQLRepository) Delete(id int) error {
	result, err := r.db.Exec(`DELETE FROM spaces WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(result, "space", id)
}
