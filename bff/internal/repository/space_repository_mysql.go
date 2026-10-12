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
	rows, err := r.db.Query(`SELECT id, title, space_key FROM spaces ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	spaces := []domain.Space{}
	for rows.Next() {
		var space domain.Space
		// CSV から取り込んだ直後などはキーがない（次の Migrate で付く）
		var key sql.NullString
		if err := rows.Scan(&space.ID, &space.Title, &key); err != nil {
			return nil, err
		}
		space.Key = key.String
		spaces = append(spaces, space)
	}
	return spaces, rows.Err()
}

// Create はスペースを追加する。キーが空なら NULL（キーなし）にする
func (r *spaceMySQLRepository) Create(space domain.Space) error {
	key := sql.NullString{String: space.Key, Valid: space.Key != ""}
	_, err := r.db.Exec(`INSERT INTO spaces (title, space_key) VALUES (?, ?)`, space.Title, key)
	return err
}

// Update はスペースの名前とキーを変える。課題キーはスペースのキーから作るので、属するタスクのキーも変わる
func (r *spaceMySQLRepository) Update(id int, space domain.Space) error {
	key := sql.NullString{String: space.Key, Valid: space.Key != ""}
	result, err := r.db.Exec(`UPDATE spaces SET title = ?, space_key = ? WHERE id = ?`, space.Title, key, id)
	if err != nil {
		return err
	}
	return requireAffected(result, "space", id)
}

// Delete はスペースと、そのスプリント（完了したときの記録を含む）・バージョン・コンポーネント（課題への付与を含む）を削除する。
// 課題のスプリントは、課題が未割り当てへ移るとき（todos の Update）にバックログへ戻る
func (r *spaceMySQLRepository) Delete(id int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM sprint_issues WHERE sprint_id IN (SELECT id FROM sprints WHERE space_id = ?)`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM sprints WHERE space_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM todo_versions WHERE version_id IN (SELECT id FROM versions WHERE space_id = ?)`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM versions WHERE space_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM todo_components WHERE component_id IN (SELECT id FROM components WHERE space_id = ?)`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM components WHERE space_id = ?`, id); err != nil {
		return err
	}
	result, err := tx.Exec(`DELETE FROM spaces WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if err := requireAffected(result, "space", id); err != nil {
		return err
	}
	return tx.Commit()
}
