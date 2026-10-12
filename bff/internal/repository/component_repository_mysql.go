package repository

import (
	"database/sql"
	"errors"

	"todo-api/internal/domain"
)

type componentMySQLRepository struct {
	db *sql.DB
}

func NewComponentMySQLRepository(db *sql.DB) ComponentRepository {
	return &componentMySQLRepository{db: db}
}

const selectComponents = `SELECT c.id, c.space_id, c.name, c.description, c.default_assignee_id, u.display_name, u.avatar_color, c.created_at
	FROM components c LEFT JOIN users u ON u.id = c.default_assignee_id`

func (r *componentMySQLRepository) FindAll() ([]domain.Component, error) {
	rows, err := r.db.Query(selectComponents + ` ORDER BY c.space_id, LOWER(c.name), c.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	components := []domain.Component{}
	for rows.Next() {
		component, err := scanComponent(rows)
		if err != nil {
			return nil, err
		}
		components = append(components, component)
	}
	return components, rows.Err()
}

func (r *componentMySQLRepository) FindByID(id int) (*domain.Component, error) {
	component, err := scanComponent(r.db.QueryRow(selectComponents+` WHERE c.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &component, nil
}

func scanComponent(row interface{ Scan(dest ...any) error }) (domain.Component, error) {
	var c domain.Component
	var assignee nullUser
	if err := row.Scan(&c.ID, &c.SpaceID, &c.Name, &c.Description, &c.DefaultAssigneeID, &assignee.name, &assignee.color, &c.CreatedAt); err != nil {
		return domain.Component{}, err
	}
	c.DefaultAssignee = assignee.summary(c.DefaultAssigneeID)
	return c, nil
}

func (r *componentMySQLRepository) Create(component domain.Component) (int, error) {
	result, err := r.db.Exec(
		`INSERT INTO components (space_id, name, description, default_assignee_id, created_at) VALUES (?, ?, ?, ?, ?)`,
		component.SpaceID, component.Name, component.Description, component.DefaultAssigneeID, currentTime())
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (r *componentMySQLRepository) Update(component domain.Component) error {
	result, err := r.db.Exec(
		`UPDATE components SET name = ?, description = ?, default_assignee_id = ? WHERE id = ?`,
		component.Name, component.Description, component.DefaultAssigneeID, component.ID)
	if err != nil {
		return err
	}
	return requireAffected(result, "component", component.ID)
}

func (r *componentMySQLRepository) Delete(id int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM todo_components WHERE component_id = ?`, id); err != nil {
		return err
	}
	result, err := tx.Exec(`DELETE FROM components WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if err := requireAffected(result, "component", id); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *componentMySQLRepository) SetTodoComponents(todoID int, componentIDs []int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM todo_components WHERE todo_id = ?`, todoID); err != nil {
		return err
	}
	for _, componentID := range componentIDs {
		if _, err := tx.Exec(`INSERT INTO todo_components (todo_id, component_id) VALUES (?, ?)`, todoID, componentID); err != nil {
			return err
		}
	}
	return tx.Commit()
}
