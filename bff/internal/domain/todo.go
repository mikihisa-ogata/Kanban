package domain

import "time"

type Status string

const (
	StatusOpen       Status = "Open"
	StatusInProgress Status = "InProgress"
	StatusWaiting    Status = "Waiting"
	StatusClosed     Status = "Closed"
)

type Todo struct {
	ID          int
	Title       string
	Done        bool
	Deadline    string
	Status      Status
	EpicID      int       // 0 の場合はエピック未割り当て
	Description string    // 任意。空文字の場合は説明なし
	SpaceID     int       // 0 の場合はスペース未割り当て。エピックに紐付く場合はエピックと同じスペース
	CreatedAt   time.Time // 作成日時（UTC）
	UpdatedAt   time.Time // 最終更新日時（UTC）。並び替えでは変わらない
}
