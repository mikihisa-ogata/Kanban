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

// Comment は課題へのコメント
type Comment struct {
	ID     int
	TodoID int
	// AuthorID は投稿者のユーザー ID。0 はユーザー機能の追加前のコメント（投稿者なし）
	AuthorID int
	// Author は投稿者の概要。投稿者なしなら nil。API の出力専用
	Author    *UserSummary
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

// CanBeChangedBy は userID のユーザーがコメントを編集・削除できるかを返す。
// 投稿者本人だけができる。投稿者のないコメント（ユーザー機能の追加前のもの）は誰でもできる
func (c Comment) CanBeChangedBy(userID int) bool {
	return c.AuthorID == 0 || c.AuthorID == userID
}
