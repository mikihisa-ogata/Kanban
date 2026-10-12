package repository

import (
	"database/sql"
	"errors"

	"todo-api/internal/domain"
)

type commentMySQLRepository struct {
	db *sql.DB
}

func NewCommentMySQLRepository(db *sql.DB) CommentRepository {
	return &commentMySQLRepository{db: db}
}

// selectComments はコメントを読む SELECT。投稿者の表示名と色も読む（scanComment と列を合わせる）
const selectComments = `SELECT c.id, c.todo_id, c.body, c.created_at, c.updated_at, c.author_id, u.display_name, u.avatar_color
		 FROM comments c LEFT JOIN users u ON u.id = c.author_id`

func (r *commentMySQLRepository) FindByTodo(todoID int) ([]domain.Comment, error) {
	rows, err := r.db.Query(selectComments+` WHERE c.todo_id = ? ORDER BY c.created_at, c.id`, todoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	comments := []domain.Comment{}
	for rows.Next() {
		comment, err := scanComment(rows)
		if err != nil {
			return nil, err
		}
		comments = append(comments, comment)
	}
	return comments, rows.Err()
}

func (r *commentMySQLRepository) FindByID(id int) (*domain.Comment, error) {
	comment, err := scanComment(r.db.QueryRow(selectComments+` WHERE c.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

func scanComment(row interface{ Scan(dest ...any) error }) (domain.Comment, error) {
	var comment domain.Comment
	var updatedAt sql.NullTime
	var author nullUser
	if err := row.Scan(&comment.ID, &comment.TodoID, &comment.Body, &comment.CreatedAt, &updatedAt,
		&comment.AuthorID, &author.name, &author.color); err != nil {
		return domain.Comment{}, err
	}
	comment.Author = author.summary(comment.AuthorID)
	if updatedAt.Valid {
		comment.UpdatedAt = &updatedAt.Time
	}
	return comment, nil
}

func (r *commentMySQLRepository) Create(comment domain.Comment) (int, error) {
	result, err := r.db.Exec(
		`INSERT INTO comments (todo_id, body, author_id, created_at) VALUES (?, ?, ?, ?)`,
		comment.TodoID, comment.Body, comment.AuthorID, currentTime())
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (r *commentMySQLRepository) Update(id int, body string) error {
	result, err := r.db.Exec(`UPDATE comments SET body = ?, updated_at = ? WHERE id = ?`, body, currentTime(), id)
	if err != nil {
		return err
	}
	return requireAffected(result, "comment", id)
}

func (r *commentMySQLRepository) Delete(id int) error {
	result, err := r.db.Exec(`DELETE FROM comments WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return requireAffected(result, "comment", id)
}
