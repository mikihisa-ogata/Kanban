package repository

import (
	"database/sql"

	"todo-api/internal/domain"
)

type issueLinkMySQLRepository struct {
	db *sql.DB
}

func NewIssueLinkMySQLRepository(db *sql.DB) IssueLinkRepository {
	return &issueLinkMySQLRepository{db: db}
}

func (r *issueLinkMySQLRepository) FindAll() ([]domain.Link, error) {
	rows, err := r.db.Query(`SELECT id, link_type, source_id, target_id FROM issue_links ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	links := []domain.Link{}
	for rows.Next() {
		var link domain.Link
		var linkType string
		if err := rows.Scan(&link.ID, &linkType, &link.SourceID, &link.TargetID); err != nil {
			return nil, err
		}
		link.Type = domain.LinkType(linkType)
		links = append(links, link)
	}
	return links, rows.Err()
}

func (r *issueLinkMySQLRepository) Create(link domain.Link) (int, error) {
	result, err := r.db.Exec(
		`INSERT INTO issue_links (link_type, source_id, target_id, created_at) VALUES (?, ?, ?, ?)`,
		string(link.Type), link.SourceID, link.TargetID, currentTime())
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (r *issueLinkMySQLRepository) Delete(id int) error {
	result, err := r.db.Exec(`DELETE FROM issue_links WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(result, "issue link", id)
}
