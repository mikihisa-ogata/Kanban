package domain

import (
	"math"
	"time"
)

type Status string

const (
	StatusOpen       Status = "Open"
	StatusInProgress Status = "InProgress"
	StatusWaiting    Status = "Waiting"
	StatusClosed     Status = "Closed"
)

// IssueType は課題タイプ（Jira の Task / Bug / Story）
type IssueType string

const (
	IssueTypeTask  IssueType = "Task"
	IssueTypeBug   IssueType = "Bug"
	IssueTypeStory IssueType = "Story"
	// IssueTypeSubtask はサブタスク。必ず親課題（ParentID）を持つ
	IssueTypeSubtask IssueType = "Subtask"
)

// ValidIssueType は課題タイプとして使える値かを返す
func ValidIssueType(t IssueType) bool {
	return t == IssueTypeTask || t == IssueTypeBug || t == IssueTypeStory || t == IssueTypeSubtask
}

// Priority は優先度（Jira の Highest / High / Medium / Low / Lowest）
type Priority string

const (
	PriorityHighest Priority = "Highest"
	PriorityHigh    Priority = "High"
	PriorityMedium  Priority = "Medium"
	PriorityLow     Priority = "Low"
	PriorityLowest  Priority = "Lowest"
)

// ValidPriority は優先度として使える値かを返す
func ValidPriority(p Priority) bool {
	switch p {
	case PriorityHighest, PriorityHigh, PriorityMedium, PriorityLow, PriorityLowest:
		return true
	}
	return false
}

// storyPointsMax はストーリーポイントの上限（DB の DECIMAL(8,2) に収まる値）
const storyPointsMax = 999999.99

// ValidStoryPoints はストーリーポイントとして使える値かを返す。0 以上、小数は第 2 位まで
func ValidStoryPoints(points float64) bool {
	if math.IsNaN(points) || points < 0 || points > storyPointsMax {
		return false
	}
	scaled := points * 100
	return math.Abs(scaled-math.Round(scaled)) < 1e-6
}

type Todo struct {
	ID          int
	Title       string
	Done        bool
	Deadline    string
	Status      Status
	IssueType   IssueType // 課題タイプ。既存のタスクは Task
	Priority    Priority  // 優先度。既定と既存のタスクは Medium
	Labels      []string  // ラベル（名前順）。ラベルがなければ空の配列
	StoryPoints *float64  // ストーリーポイント（見積もり）。nil は未設定
	ParentID    int       // サブタスクの親課題の ID。0 はサブタスクではない
	EpicID      int       // 0 の場合はエピック未割り当て
	Description string    // 任意。空文字の場合は説明なし
	SpaceID     int       // 0 の場合はスペース未割り当て。エピックに紐付く場合はエピックと同じスペース
	Number      int       // スペース内の連番。スペースを移ると移動先で採番し直す。削除しても再利用しない
	Key         string    // 課題キー（例: KAN-12）。未割り当てのタスクは TASK-3 のようになる。API の出力専用
	CreatedAt   time.Time // 作成日時（UTC）
	UpdatedAt   time.Time // 最終更新日時（UTC）。並び替えでは変わらない
}
