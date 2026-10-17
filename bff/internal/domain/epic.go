package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
)

type Epic struct {
	ID      int
	Title   string
	SpaceID int // 0 の場合はスペース未割り当て
	// StartDate は開始日、DueDate は期限日（例: 2026-11-01）。空文字は未設定。タイムラインのバーの期間になる
	StartDate string
	DueDate   string
	// Color はタイムラインのバーの色（#RRGGBB）。空文字は既定の色
	Color     string
	CreatedAt time.Time // 作成日時（UTC）
	UpdatedAt time.Time // 最終更新日時（UTC）
}

var ErrInvalidEpicDates = errors.New("日付は YYYY-MM-DD 形式で、期限日を開始日以降にしてください")

var ErrInvalidEpicColor = errors.New("色は #RRGGBB 形式で指定してください")

var epicColorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// ValidateEpicDates は開始日・期限日を検証する。どちらも空にでき、両方あるときは期限日を開始日以降にする
func ValidateEpicDates(startDate, dueDate string) error {
	var start, due time.Time
	var err error
	if startDate != "" {
		if start, err = time.Parse(sprintDateLayout, startDate); err != nil {
			return ErrInvalidEpicDates
		}
	}
	if dueDate != "" {
		if due, err = time.Parse(sprintDateLayout, dueDate); err != nil {
			return ErrInvalidEpicDates
		}
	}
	if startDate != "" && dueDate != "" && due.Before(start) {
		return ErrInvalidEpicDates
	}
	return nil
}

// NormalizeEpicColor は色を大文字の #RRGGBB にして返す。空文字は既定の色のまま。形式が違えば ErrInvalidEpicColor
func NormalizeEpicColor(color string) (string, error) {
	color = strings.TrimSpace(color)
	if color == "" {
		return "", nil
	}
	if !epicColorPattern.MatchString(color) {
		return "", ErrInvalidEpicColor
	}
	return strings.ToUpper(color), nil
}
