package repository

import (
	"database/sql"
	"errors"

	"todo-api/internal/domain"
)

type versionMySQLRepository struct {
	db *sql.DB
}

func NewVersionMySQLRepository(db *sql.DB) VersionRepository {
	return &versionMySQLRepository{db: db}
}

const selectVersions = `SELECT id, space_id, name, description, status, start_date, release_date, created_at FROM versions`

func (r *versionMySQLRepository) FindAll() ([]domain.Version, error) {
	rows, err := r.db.Query(selectVersions + ` ORDER BY space_id, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	versions := []domain.Version{}
	for rows.Next() {
		version, err := scanVersion(rows)
		if err != nil {
			return nil, err
		}
		versions = append(versions, version)
	}
	return versions, rows.Err()
}

func (r *versionMySQLRepository) FindByID(id int) (*domain.Version, error) {
	version, err := scanVersion(r.db.QueryRow(selectVersions+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &version, nil
}

func scanVersion(row interface{ Scan(dest ...any) error }) (domain.Version, error) {
	var v domain.Version
	var status string
	var start, release sql.NullTime
	if err := row.Scan(&v.ID, &v.SpaceID, &v.Name, &v.Description, &status, &start, &release, &v.CreatedAt); err != nil {
		return domain.Version{}, err
	}
	v.Status = domain.VersionStatus(status)
	if start.Valid {
		v.StartDate = start.Time.Format(deadlineLayout)
	}
	if release.Valid {
		v.ReleaseDate = release.Time.Format(deadlineLayout)
	}
	return v, nil
}

func (r *versionMySQLRepository) Create(version domain.Version) (int, error) {
	start, err := toNullDeadline(version.StartDate)
	if err != nil {
		return 0, err
	}
	release, err := toNullDeadline(version.ReleaseDate)
	if err != nil {
		return 0, err
	}
	result, err := r.db.Exec(
		`INSERT INTO versions (space_id, name, description, status, start_date, release_date, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		version.SpaceID, version.Name, version.Description, string(version.Status), start, release, currentTime())
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (r *versionMySQLRepository) Update(version domain.Version) error {
	start, err := toNullDeadline(version.StartDate)
	if err != nil {
		return err
	}
	release, err := toNullDeadline(version.ReleaseDate)
	if err != nil {
		return err
	}
	result, err := r.db.Exec(
		`UPDATE versions SET name = ?, description = ?, status = ?, start_date = ?, release_date = ? WHERE id = ?`,
		version.Name, version.Description, string(version.Status), start, release, version.ID)
	if err != nil {
		return err
	}
	return requireAffected(result, "version", version.ID)
}

func (r *versionMySQLRepository) Release(id int, releaseDate string, moveIDs []int, moveTo int) error {
	release, err := toNullDeadline(releaseDate)
	if err != nil {
		return err
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, todoID := range moveIDs {
		if _, err := tx.Exec(`DELETE FROM todo_versions WHERE todo_id = ? AND version_id = ?`, todoID, id); err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT IGNORE INTO todo_versions (todo_id, version_id) VALUES (?, ?)`, todoID, moveTo); err != nil {
			return err
		}
	}
	result, err := tx.Exec(`UPDATE versions SET status = ?, release_date = ? WHERE id = ?`, string(domain.VersionReleased), release, id)
	if err != nil {
		return err
	}
	if err := requireAffected(result, "version", id); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *versionMySQLRepository) Delete(id int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM todo_versions WHERE version_id = ?`, id); err != nil {
		return err
	}
	result, err := tx.Exec(`DELETE FROM versions WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if err := requireAffected(result, "version", id); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *versionMySQLRepository) SetTodoVersions(todoID int, versionIDs []int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM todo_versions WHERE todo_id = ?`, todoID); err != nil {
		return err
	}
	for _, versionID := range versionIDs {
		if _, err := tx.Exec(`INSERT INTO todo_versions (todo_id, version_id) VALUES (?, ?)`, todoID, versionID); err != nil {
			return err
		}
	}
	return tx.Commit()
}
