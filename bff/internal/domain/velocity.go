package domain

import (
	"errors"
	"math"
	"sort"
	"time"
)

// VelocitySprintCount はベロシティに並べる完了したスプリントの数（Jira と同じく直近 7 つ）
const VelocitySprintCount = 7

// VelocitySprint はベロシティの 1 スプリント分。Commitment は開始したときにスプリントに入っていた課題（サブタスクを除く。
// 開始前に完了していたものも含む）のストーリーポイントの合計、Completed は完了したときに完了していた課題のポイントの合計（未設定は 0）
type VelocitySprint struct {
	SprintID    int
	Name        string
	StartDate   string
	EndDate     string
	CompletedAt *time.Time
	Commitment  float64
	Completed   float64
}

// Velocity はスペースのベロシティ。Sprints は完了した直近のスプリントを古い順に並べたもの、Average はその Completed の平均（なければ 0）
type Velocity struct {
	Sprints []VelocitySprint
	Average float64
}

// VelocityInput はベロシティの集計に使うもの
type VelocityInput struct {
	// Sprints はスペースのスプリント（完了していないものは無視する）
	Sprints []Sprint
	// Todos は今の課題。Recorded はスプリントの ID ごとの、完了したときに記録した課題
	Todos    []Todo
	Recorded map[int][]SprintIssue
	// History は sprint・space・statusCategory・done・storyPoints の変更履歴
	History  []HistoryEntry
	Location *time.Location
}

// ComputeVelocity は完了した直近 VelocitySprintCount 個のスプリントの、コミットメントと完了したポイントを返す。
// コミットメントは変更履歴から開始した時点の状態を再計算する（バーンダウンと同じ考え方）。完了は完了したときの記録を使う
func ComputeVelocity(in VelocityInput) (Velocity, error) {
	loc := in.Location
	if loc == nil {
		loc = time.UTC
	}
	closed := []Sprint{}
	for _, sprint := range in.Sprints {
		if sprint.State == SprintStateClosed {
			closed = append(closed, sprint)
		}
	}
	// 完了した順。完了した日時がなければ（記録の前のもの）ID 順で先に置く
	sort.SliceStable(closed, func(a, b int) bool {
		ca, cb := closed[a].CompletedAt, closed[b].CompletedAt
		switch {
		case ca == nil && cb == nil:
			return closed[a].ID < closed[b].ID
		case ca == nil || cb == nil:
			return ca == nil
		case !ca.Equal(*cb):
			return ca.Before(*cb)
		}
		return closed[a].ID < closed[b].ID
	})
	if len(closed) > VelocitySprintCount {
		closed = closed[len(closed)-VelocitySprintCount:]
	}

	result := Velocity{Sprints: []VelocitySprint{}}
	total := 0.0
	for _, sprint := range closed {
		recorded := in.Recorded[sprint.ID]
		commitment, err := sprintCommitment(BurndownInput{Sprint: sprint, Todos: in.Todos, Recorded: recorded, History: in.History, Location: loc})
		if err != nil {
			return Velocity{}, err
		}
		completed := 0.0
		for _, issue := range recorded {
			if issue.Completed && issue.StoryPoints != nil {
				completed += *issue.StoryPoints
			}
		}
		completed = roundPoints(completed)
		total += completed
		result.Sprints = append(result.Sprints, VelocitySprint{
			SprintID: sprint.ID, Name: sprint.Name, StartDate: sprint.StartDate, EndDate: sprint.EndDate, CompletedAt: sprint.CompletedAt,
			Commitment: commitment, Completed: completed,
		})
	}
	if len(result.Sprints) > 0 {
		result.Average = roundPoints(total / float64(len(result.Sprints)))
	}
	return result, nil
}

// sprintCommitment は開始した時点でスプリントに入っていた課題（サブタスクを除く）のストーリーポイントの合計を返す。
// 期間がなく起点が分からないスプリントは 0
func sprintCommitment(in BurndownInput) (float64, error) {
	start, _, _, err := sprintPeriod(in.Sprint, in.Location)
	if errors.Is(err, ErrSprintNotStarted) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	total := 0.0
	for _, issue := range burndownIssues(in) {
		if issue.todo.CreatedAt.After(start) || valueAt(issue.steps, valueMember, start, boolValue(issue.member)) != 1 {
			continue
		}
		if points := valueAt(issue.steps, valuePoints, start, issue.points); !math.IsNaN(points) {
			total += points
		}
	}
	return roundPoints(total), nil
}

// roundPoints はストーリーポイントを小数第 2 位で丸める（小数の足し算の誤差を消す）
func roundPoints(p float64) float64 {
	return math.Round(p*100) / 100
}
