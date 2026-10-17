package repository

import (
	"database/sql"
	"encoding/json"
	"errors"

	"todo-api/internal/domain"
)

type dashboardMySQLRepository struct {
	db *sql.DB
}

func NewDashboardMySQLRepository(db *sql.DB) DashboardRepository {
	return &dashboardMySQLRepository{db: db}
}

// storedGadget は dashboards.gadgets の JSON の 1 要素
type storedGadget struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	FilterID int    `json:"filterId,omitempty"`
}

func (r *dashboardMySQLRepository) FindByUser(userID int) ([]domain.DashboardGadget, error) {
	var raw string
	err := r.db.QueryRow(`SELECT gadgets FROM dashboards WHERE user_id = ?`, userID).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var stored []storedGadget
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		return nil, err
	}
	gadgets := make([]domain.DashboardGadget, len(stored))
	for i, g := range stored {
		gadgets[i] = domain.DashboardGadget{ID: g.ID, Type: domain.GadgetType(g.Type), FilterID: g.FilterID}
	}
	return gadgets, nil
}

func (r *dashboardMySQLRepository) Save(userID int, gadgets []domain.DashboardGadget) error {
	stored := make([]storedGadget, len(gadgets))
	for i, g := range gadgets {
		stored[i] = storedGadget{ID: g.ID, Type: string(g.Type), FilterID: g.FilterID}
	}
	raw, err := json.Marshal(stored)
	if err != nil {
		return err
	}
	now := currentTime()
	_, err = r.db.Exec(
		`INSERT INTO dashboards (user_id, gadgets, updated_at) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE gadgets = ?, updated_at = ?`,
		userID, string(raw), now, string(raw), now)
	return err
}
