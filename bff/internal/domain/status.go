package domain

import (
	"errors"
	"strings"
	"unicode/utf8"
)

// StatusCategory はステータスのカテゴリ（Jira の To Do / In Progress / Done）。
// Done のステータスにした課題は完了扱い（Todo.Done が true）になる
type StatusCategory string

const (
	StatusCategoryToDo       StatusCategory = "ToDo"
	StatusCategoryInProgress StatusCategory = "InProgress"
	StatusCategoryDone       StatusCategory = "Done"
)

// ValidStatusCategory はカテゴリとして使える値かを返す
func ValidStatusCategory(c StatusCategory) bool {
	return c == StatusCategoryToDo || c == StatusCategoryInProgress || c == StatusCategoryDone
}

// WorkflowStatus はスペースのワークフローのステータス（ボードの列）。スペース未割り当ての課題は SpaceID 0 のステータスを使う
type WorkflowStatus struct {
	ID       int
	SpaceID  int
	Name     string
	Category StatusCategory
	Position int // スペース内の並び順（ボードの列の順）
	// LegacyKey はワークフローの導入前の固定のステータス（Open など）。既定の 4 ステータスだけが持ち、追加したステータスは空。
	// API の Todo.Status（互換用）と、status 文字列での指定に使う
	LegacyKey Status `json:"-"`
}

// DefaultWorkflow はスペースを作ったときの既定のステータス（導入前の 4 ステータス）を並び順に返す
func DefaultWorkflow() []WorkflowStatus {
	return []WorkflowStatus{
		{Name: "オープン", Category: StatusCategoryToDo, Position: 1, LegacyKey: StatusOpen},
		{Name: "進行中", Category: StatusCategoryInProgress, Position: 2, LegacyKey: StatusInProgress},
		{Name: "待ち状況", Category: StatusCategoryInProgress, Position: 3, LegacyKey: StatusWaiting},
		{Name: "クローズ", Category: StatusCategoryDone, Position: 4, LegacyKey: StatusClosed},
	}
}

// LegacyCategory は導入前の固定のステータスのカテゴリを返す。知らない値なら false
func LegacyCategory(s Status) (StatusCategory, bool) {
	switch s {
	case StatusOpen:
		return StatusCategoryToDo, true
	case StatusInProgress, StatusWaiting:
		return StatusCategoryInProgress, true
	case StatusClosed:
		return StatusCategoryDone, true
	}
	return "", false
}

// LegacyStatus は互換用の Todo.Status の値を返す。既定のステータスはその値、追加したステータスはカテゴリから決める
func (s WorkflowStatus) LegacyStatus() Status {
	if s.LegacyKey != "" {
		return s.LegacyKey
	}
	switch s.Category {
	case StatusCategoryInProgress:
		return StatusInProgress
	case StatusCategoryDone:
		return StatusClosed
	}
	return StatusOpen
}

// MatchStatus は別のワークフローへ移るときの対応先を、statuses（移動先のステータス。並び順）から選ぶ。
// 同じ名前があればそれ、なければ同じカテゴリの先頭、それもなければ先頭のステータス。statuses が空なら false
func MatchStatus(statuses []WorkflowStatus, from WorkflowStatus) (WorkflowStatus, bool) {
	for _, s := range statuses {
		if s.Name == from.Name {
			return s, true
		}
	}
	return FirstInCategory(statuses, from.Category)
}

// FirstInCategory はカテゴリの先頭のステータスを返す。そのカテゴリがなければ先頭のステータス。statuses が空なら false
func FirstInCategory(statuses []WorkflowStatus, category StatusCategory) (WorkflowStatus, bool) {
	for _, s := range statuses {
		if s.Category == category {
			return s, true
		}
	}
	if len(statuses) == 0 {
		return WorkflowStatus{}, false
	}
	return statuses[0], true
}

// statusNameMax はステータス名の最大文字数（DB の VARCHAR(50)）
const statusNameMax = 50

var ErrInvalidStatusName = errors.New("ステータスの名前は 1〜50 文字で入力してください")

// NormalizeStatusName は前後の空白を除いたステータス名を返す。空や長すぎる名前は ErrInvalidStatusName
func NormalizeStatusName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > statusNameMax {
		return "", ErrInvalidStatusName
	}
	return name, nil
}
