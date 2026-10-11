package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// commentBodyMax はコメント本文の最大文字数
const commentBodyMax = 10000

var ErrInvalidComment = errors.New("コメントは空にできず、10000 文字以内にしてください")

// Comment は課題へのコメント。投稿者はユーザー機能の追加後に持たせる
type Comment struct {
	ID        int
	TodoID    int
	Body      string     // Markdown
	CreatedAt time.Time  // 投稿日時（UTC）
	UpdatedAt *time.Time // 最後に編集した日時（UTC）。nil は編集していない
}

// ValidateCommentBody はコメント本文として使えるかを確かめる。空白だけの本文と長すぎる本文は ErrInvalidComment
func ValidateCommentBody(body string) error {
	if strings.TrimSpace(body) == "" || utf8.RuneCountInString(body) > commentBodyMax {
		return ErrInvalidComment
	}
	return nil
}
