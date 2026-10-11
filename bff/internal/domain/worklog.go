package domain

import (
	"errors"
	"time"
	"unicode/utf8"
)

// worklogNoteMax はメモの最大文字数
const worklogNoteMax = 10000

var ErrInvalidWorklogNote = errors.New("メモは 10000 文字以内にしてください")

// Worklog は課題の作業ログ（作業を記録した 1 件）
type Worklog struct {
	ID        int
	TodoID    int
	Minutes   int        // 作業時間（分）
	StartedAt time.Time  // 作業の開始日時（UTC）
	Note      string     // メモ。空文字はメモなし
	CreatedAt time.Time  // 記録した日時（UTC）
	UpdatedAt *time.Time // 最後に編集した日時（UTC）。nil は編集していない
}

// ValidateWorklogNote はメモとして使えるかを確かめる
func ValidateWorklogNote(note string) error {
	if utf8.RuneCountInString(note) > worklogNoteMax {
		return ErrInvalidWorklogNote
	}
	return nil
}
