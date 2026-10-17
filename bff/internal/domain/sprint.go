package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// SprintState はスプリントの状態（Jira の future / active / closed）
type SprintState string

const (
	// SprintStateFuture は作成しただけで、まだ開始していないスプリント
	SprintStateFuture SprintState = "future"
	// SprintStateActive は開始したスプリント。スペースごとに同時に 1 つまで
	SprintStateActive SprintState = "active"
	// SprintStateClosed は完了したスプリント
	SprintStateClosed SprintState = "closed"
)

// Sprint はスペースのスプリント。課題は Todo.SprintID でスプリントに入る（0 はバックログ）
type Sprint struct {
	ID      int
	SpaceID int
	Name    string
	Goal    string // スプリントのゴール。空文字はなし
	State   SprintState
	// StartDate・EndDate は期間（例: 2026-10-12）。開始前は空文字のこともある
	StartDate string
	EndDate   string
	// StartedAt は開始した日時（UTC）。未開始か、開始した日時を記録する前に開始したスプリントは nil
	StartedAt *time.Time
	// CompletedAt は完了した日時（UTC）。完了していなければ nil
	CompletedAt *time.Time
	CreatedAt   time.Time
	// CompletedIssues・IncompleteIssues・CompletedPoints は完了したときに記録した課題の件数と、完了した課題のストーリーポイントの合計。
	// 完了したスプリントだけが持つ（それ以外は 0）。API の出力専用
	CompletedIssues  int
	IncompleteIssues int
	CompletedPoints  float64
}

// SprintIssue はスプリントを完了したときに記録した課題（サブタスクを除く）。
// StoryPoints はその時点のストーリーポイント（nil は未設定）、Completed は完了（Done カテゴリ）だったか
type SprintIssue struct {
	SprintID    int
	TodoID      int
	StoryPoints *float64
	Completed   bool
}

// sprintNameMax はスプリント名の最大文字数（DB の VARCHAR(100)）
const sprintNameMax = 100

// sprintGoalMax はスプリントのゴールの最大文字数
const sprintGoalMax = 1000

// sprintDateLayout はスプリントの期間の日付の形式
const sprintDateLayout = "2006-01-02"

var ErrInvalidSprintName = errors.New("スプリントの名前は 1〜100 文字で入力してください")

var ErrInvalidSprintGoal = errors.New("スプリントのゴールは 1000 文字以内で入力してください")

var ErrInvalidSprintDates = errors.New("スプリントの期間は YYYY-MM-DD 形式で、終了日を開始日以降にしてください")

// NormalizeSprintName は前後の空白を除いたスプリント名を返す。空や長すぎる名前は ErrInvalidSprintName
func NormalizeSprintName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > sprintNameMax {
		return "", ErrInvalidSprintName
	}
	return name, nil
}

// NormalizeSprintGoal は前後の空白を除いたゴールを返す。長すぎるゴールは ErrInvalidSprintGoal
func NormalizeSprintGoal(goal string) (string, error) {
	goal = strings.TrimSpace(goal)
	if utf8.RuneCountInString(goal) > sprintGoalMax {
		return "", ErrInvalidSprintGoal
	}
	return goal, nil
}

// ValidateSprintDates はスプリントの期間を検証する。required なら両方が必要（開始するとき）。
// 片方だけの指定、形式の誤り、終了日が開始日より前なら ErrInvalidSprintDates
func ValidateSprintDates(start, end string, required bool) error {
	if start == "" && end == "" {
		if required {
			return ErrInvalidSprintDates
		}
		return nil
	}
	s, err := time.Parse(sprintDateLayout, start)
	if err != nil {
		return ErrInvalidSprintDates
	}
	e, err := time.Parse(sprintDateLayout, end)
	if err != nil {
		return ErrInvalidSprintDates
	}
	if e.Before(s) {
		return ErrInvalidSprintDates
	}
	return nil
}
