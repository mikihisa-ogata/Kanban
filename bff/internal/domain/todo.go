package domain

import "time"

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
)

// ValidIssueType は課題タイプとして使える値かを返す
func ValidIssueType(t IssueType) bool {
	return t == IssueTypeTask || t == IssueTypeBug || t == IssueTypeStory
}

type Todo struct {
	ID          int
	Title       string
	Done        bool
	Deadline    string
	Status      Status
	IssueType   IssueType // 課題タイプ。既存のタスクは Task
	EpicID      int       // 0 の場合はエピック未割り当て
	Description string    // 任意。空文字の場合は説明なし
	SpaceID     int       // 0 の場合はスペース未割り当て。エピックに紐付く場合はエピックと同じスペース
	Number      int       // スペース内の連番。スペースを移ると移動先で採番し直す。削除しても再利用しない
	Key         string    // 課題キー（例: KAN-12）。未割り当てのタスクは TASK-3 のようになる。API の出力専用
	CreatedAt   time.Time // 作成日時（UTC）
	UpdatedAt   time.Time // 最終更新日時（UTC）。並び替えでは変わらない
}
