package domain

type Status string

const (
	StatusOpen       Status = "Open"
	StatusInProgress Status = "InProgress"
	StatusWaiting    Status = "Waiting"
	StatusClosed     Status = "Closed"
)

type Todo struct {
	ID       int
	Title    string
	Done     bool
	Deadline string
	Status   Status
}
