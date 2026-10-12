package repository

import (
	"database/sql"
	"errors"
	"time"

	"github.com/go-sql-driver/mysql"

	"todo-api/internal/domain"
)

// mysqlErrDuplicateEntry は一意インデックスの重複を表す MySQL のエラー番号
const mysqlErrDuplicateEntry = 1062

type userMySQLRepository struct {
	db *sql.DB
}

func NewUserMySQLRepository(db *sql.DB) UserRepository {
	return &userMySQLRepository{db: db}
}

const selectUsers = `SELECT id, display_name, email, password_hash, avatar_color, created_at FROM users`

func (r *userMySQLRepository) Create(user domain.User) (int, error) {
	result, err := r.db.Exec(
		`INSERT INTO users (display_name, email, password_hash, avatar_color, created_at) VALUES (?, ?, ?, ?, ?)`,
		user.DisplayName, user.Email, user.PasswordHash, user.AvatarColor, currentTime())
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlErrDuplicateEntry {
		return 0, ErrDuplicateEmail
	}
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

func (r *userMySQLRepository) FindByID(id int) (*domain.User, error) {
	return findUser(r.db.QueryRow(selectUsers+` WHERE id = ?`, id))
}

func (r *userMySQLRepository) FindByEmail(email string) (*domain.User, error) {
	return findUser(r.db.QueryRow(selectUsers+` WHERE email = ?`, email))
}

func findUser(row *sql.Row) (*domain.User, error) {
	var user domain.User
	err := row.Scan(&user.ID, &user.DisplayName, &user.Email, &user.PasswordHash, &user.AvatarColor, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

type sessionMySQLRepository struct {
	db *sql.DB
}

func NewSessionMySQLRepository(db *sql.DB) SessionRepository {
	return &sessionMySQLRepository{db: db}
}

func (r *sessionMySQLRepository) Create(tokenHash string, userID int, expiresAt time.Time) error {
	_, err := r.db.Exec(
		`INSERT INTO sessions (token_hash, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		tokenHash, userID, currentTime(), expiresAt.UTC())
	return err
}

func (r *sessionMySQLRepository) FindUserID(tokenHash string) (int, error) {
	var userID int
	err := r.db.QueryRow(
		`SELECT user_id FROM sessions WHERE token_hash = ? AND expires_at > ?`, tokenHash, currentTime()).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return userID, nil
}

func (r *sessionMySQLRepository) Delete(tokenHash string) error {
	_, err := r.db.Exec(`DELETE FROM sessions WHERE token_hash = ?`, tokenHash)
	return err
}

func (r *sessionMySQLRepository) DeleteExpired() error {
	_, err := r.db.Exec(`DELETE FROM sessions WHERE expires_at <= ?`, currentTime())
	return err
}
