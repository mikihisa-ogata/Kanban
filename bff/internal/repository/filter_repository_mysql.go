package repository

import (
	"database/sql"
	"errors"
	"time"

	"todo-api/internal/domain"
)

type filterMySQLRepository struct {
	db *sql.DB
}

func NewFilterMySQLRepository(db *sql.DB) FilterRepository {
	return &filterMySQLRepository{db: db}
}

const selectFilters = `SELECT id, owner_id, name, jql, favourite, created_at, updated_at FROM saved_filters`

func (r *filterMySQLRepository) FindByOwner(ownerID int) ([]domain.SavedFilter, error) {
	rows, err := r.db.Query(selectFilters+` WHERE owner_id = ? ORDER BY LOWER(name), id`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	filters := []domain.SavedFilter{}
	for rows.Next() {
		filter, err := scanFilter(rows)
		if err != nil {
			return nil, err
		}
		filters = append(filters, filter)
	}
	return filters, rows.Err()
}

func (r *filterMySQLRepository) FindByID(id int) (*domain.SavedFilter, error) {
	filter, err := scanFilter(r.db.QueryRow(selectFilters+` WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &filter, nil
}

func scanFilter(row interface{ Scan(dest ...any) error }) (domain.SavedFilter, error) {
	var f domain.SavedFilter
	if err := row.Scan(&f.ID, &f.OwnerID, &f.Name, &f.JQL, &f.Favourite, &f.CreatedAt, &f.UpdatedAt); err != nil {
		return domain.SavedFilter{}, err
	}
	return f, nil
}

func (r *filterMySQLRepository) Create(filter domain.SavedFilter) (int, error) {
	now := currentTime()
	result, err := r.db.Exec(
		`INSERT INTO saved_filters (owner_id, name, jql, favourite, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		filter.OwnerID, filter.Name, filter.JQL, filter.Favourite, now, now)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (r *filterMySQLRepository) Update(filter domain.SavedFilter) error {
	result, err := r.db.Exec(
		`UPDATE saved_filters SET name = ?, jql = ?, favourite = ?, updated_at = ? WHERE id = ?`,
		filter.Name, filter.JQL, filter.Favourite, currentTime(), filter.ID)
	if err != nil {
		return err
	}
	return requireAffected(result, "filter", filter.ID)
}

func (r *filterMySQLRepository) Delete(id int) error {
	result, err := r.db.Exec(`DELETE FROM saved_filters WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(result, "filter", id)
}

type issueViewMySQLRepository struct {
	db *sql.DB
}

func NewIssueViewMySQLRepository(db *sql.DB) IssueViewRepository {
	return &issueViewMySQLRepository{db: db}
}

// Record は表示した順に並べられるよう、日時をミリ秒まで残す
func (r *issueViewMySQLRepository) Record(userID int, todoID int) error {
	now := time.Now().UTC().Truncate(time.Millisecond)
	_, err := r.db.Exec(
		`INSERT INTO issue_views (user_id, todo_id, viewed_at) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE viewed_at = ?`,
		userID, todoID, now, now)
	return err
}
