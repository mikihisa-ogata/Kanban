package repository

import (
	"errors"
	"time"

	"todo-api/internal/domain"
)

// ErrDuplicateEmail は同じメールアドレスのユーザーが既にあることを表す
var ErrDuplicateEmail = errors.New("duplicate email")

type UserRepository interface {
	// Create はユーザーを作成し、その ID を返す。作成日時は現在日時にする。
	// 同じメールアドレスのユーザーがあれば ErrDuplicateEmail を返す
	Create(user domain.User) (int, error)
	// FindByID はユーザーを返す。存在しなければ nil を返す
	FindByID(id int) (*domain.User, error)
	// FindByEmail はメールアドレス（正規化済み）のユーザーを返す。存在しなければ nil を返す
	FindByEmail(email string) (*domain.User, error)
}

type SessionRepository interface {
	// Create はトークンのハッシュのセッションを作る。作成日時は現在日時にする
	Create(tokenHash string, userID int, expiresAt time.Time) error
	// FindUserID はトークンのハッシュの、有効期限内のセッションのユーザー ID を返す。なければ 0 を返す
	FindUserID(tokenHash string) (int, error)
	// Delete はセッションを削除する。なくてもエラーにしない
	Delete(tokenHash string) error
	// DeleteExpired は有効期限を過ぎたセッションを削除する
	DeleteExpired() error
}
