package repository

import (
	"database/sql"

	"todo-api/internal/domain"
)

type statusMySQLRepository struct {
	db *sql.DB
}

func NewStatusMySQLRepository(db *sql.DB) StatusRepository {
	return &statusMySQLRepository{db: db}
}

const selectStatuses = `SELECT id, space_id, name, category, position, legacy_key FROM statuses`

func (r *statusMySQLRepository) FindAll() ([]domain.WorkflowStatus, error) {
	return r.query(selectStatuses + ` ORDER BY space_id, position, id`)
}

func (r *statusMySQLRepository) FindBySpace(spaceID int) ([]domain.WorkflowStatus, error) {
	return r.query(selectStatuses+` WHERE space_id = ? ORDER BY position, id`, spaceID)
}

func (r *statusMySQLRepository) query(query string, args ...any) ([]domain.WorkflowStatus, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	statuses := []domain.WorkflowStatus{}
	for rows.Next() {
		var s domain.WorkflowStatus
		var category, legacyKey string
		if err := rows.Scan(&s.ID, &s.SpaceID, &s.Name, &category, &s.Position, &legacyKey); err != nil {
			return nil, err
		}
		s.Category = domain.StatusCategory(category)
		s.LegacyKey = domain.Status(legacyKey)
		statuses = append(statuses, s)
	}
	return statuses, rows.Err()
}

// EnsureDefaults は同時に呼ばれても二重に作らないよう、スペースの行をロックしてから確かめる
func (r *statusMySQLRepository) EnsureDefaults(spaceID int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM statuses WHERE space_id = ? FOR UPDATE`, spaceID).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	for _, s := range domain.DefaultWorkflow() {
		if _, err := tx.Exec(
			`INSERT INTO statuses (space_id, name, category, position, legacy_key) VALUES (?, ?, ?, ?, ?)`,
			spaceID, s.Name, string(s.Category), s.Position, string(s.LegacyKey)); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *statusMySQLRepository) DeleteBySpace(spaceID int) error {
	_, err := r.db.Exec(`DELETE FROM statuses WHERE space_id = ?`, spaceID)
	return err
}

// Create は並び順をスペースの最後の次にして追加する
func (r *statusMySQLRepository) Create(status domain.WorkflowStatus) (int, error) {
	result, err := r.db.Exec(
		`INSERT INTO statuses (space_id, name, category, position)
		 SELECT ?, ?, ?, COALESCE(MAX(position), 0) + 1 FROM statuses WHERE space_id = ?`,
		status.SpaceID, status.Name, string(status.Category), status.SpaceID)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (r *statusMySQLRepository) Rename(id int, name string) error {
	result, err := r.db.Exec(`UPDATE statuses SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		return err
	}
	return requireAffected(result, "status", id)
}

func (r *statusMySQLRepository) Reorder(spaceID int, ids []int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for i, id := range ids {
		if _, err := tx.Exec(`UPDATE statuses SET position = ? WHERE id = ? AND space_id = ?`, i+1, id, spaceID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *statusMySQLRepository) Delete(id int) error {
	result, err := r.db.Exec(`DELETE FROM statuses WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(result, "status", id)
}
