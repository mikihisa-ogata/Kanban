package domain

import (
	"errors"
	"math"
	"sort"
	"strconv"
	"time"
)

var ErrSprintNotStarted = errors.New("バーンダウンを表示できるのは、開始中か完了したスプリントだけです")

// BurndownEventKind はバーンダウンの残りポイントを変えた出来事の種類
type BurndownEventKind string

const (
	BurndownAdded     BurndownEventKind = "added"     // スプリントに課題が追加された（作成を含む）
	BurndownRemoved   BurndownEventKind = "removed"   // スプリントから課題が除外された（スペースの移動を含む）
	BurndownCompleted BurndownEventKind = "completed" // 課題が完了（Done カテゴリ）になった
	BurndownReopened  BurndownEventKind = "reopened"  // 完了した課題が未完了に戻った
	BurndownEstimate  BurndownEventKind = "estimate"  // 未完了の課題のストーリーポイントが変わった
)

// BurndownEvent はスプリント期間中に残りポイントを変えた出来事。Change は残りポイントの増減、Remaining はその後の残りポイント。
// Points はその後の課題のストーリーポイント（未設定は nil）。追加・除外・完了・再開は、ポイントが 0 でも記録する
type BurndownEvent struct {
	Time      time.Time
	Date      string // Time の日付（YYYY-MM-DD。集計したタイムゾーン）
	TodoID    int
	Key       string
	Title     string
	Kind      BurndownEventKind
	Points    *float64
	Change    float64
	Remaining float64
}

// BurndownDay は 1 日分の残りポイントの増減。Added・Removed・Estimate・Completed はその日の種類ごとの増減の合計
// （Completed は完了で減った分と再開で増えた分の合計）。Remaining はその日の終わりの残りポイントで、まだ来ていない日は nil
type BurndownDay struct {
	Date      string
	Added     float64
	Removed   float64
	Estimate  float64
	Completed float64
	Remaining *float64
}

// Burndown はスプリントのバーンダウン。残りポイントは、スプリントに入っている未完了の課題（サブタスクを除く）の
// ストーリーポイントの合計（未設定は 0）。理想線は Start に StartPoints、End に 0 を結ぶ
type Burndown struct {
	SprintID int
	// Start は起点（開始した日時。記録がなければ開始日の 0 時）。End は計画の終わり（終了日の翌日の 0 時）。
	// Until は実績の終わり（完了した日時。開始中なら現在）
	Start       time.Time
	End         time.Time
	Until       time.Time
	StartPoints float64
	Events      []BurndownEvent
	Days        []BurndownDay
}

// BurndownInput はバーンダウンの集計に使うもの
type BurndownInput struct {
	Sprint Sprint
	// Todos は今の課題（サブタスクは集計に入れない）。Recorded は完了したときに記録したスプリントの課題
	Todos    []Todo
	Recorded []SprintIssue
	// History は sprint・space・statusCategory・done・storyPoints の変更履歴（ほかの項目は無視する）
	History  []HistoryEntry
	Now      time.Time
	Location *time.Location
}

// burndownValue は課題の状態の種類。スプリントに入っているか・完了しているか・ストーリーポイント
type burndownValue int

const (
	valueMember burndownValue = iota
	valueDone
	valuePoints
)

// burndownStep は履歴 1 件を、課題の状態の変化にしたもの。hasOld が false なら変更前の値は分からない
type burndownStep struct {
	at       time.Time
	order    int
	value    burndownValue
	hasOld   bool
	old, new float64
}

// burndownIssue は集計中の課題の状態。points は未設定なら NaN
type burndownIssue struct {
	todo         Todo
	exists       bool
	member, done bool
	points       float64
	steps        []burndownStep
}

func (i *burndownIssue) inSprint() bool { return i.exists && i.member }

func (i *burndownIssue) remaining() float64 {
	if !i.inSprint() || i.done || math.IsNaN(i.points) {
		return 0
	}
	return i.points
}

func (i *burndownIssue) pointsValue() *float64 {
	if math.IsNaN(i.points) {
		return nil
	}
	p := i.points
	return &p
}

// ComputeBurndown は変更履歴から、スプリント期間中の課題の追加・除外・完了・ポイント変更を再計算してバーンダウンを返す。
// 履歴がない課題は、今（完了したスプリントでは記録した課題も）スプリントに入っていれば最初から入っていたものとし、
// 履歴がない項目は今の値のままだったものとする。未開始のスプリントは ErrSprintNotStarted
func ComputeBurndown(in BurndownInput) (Burndown, error) {
	sprint := in.Sprint
	loc := in.Location
	if loc == nil {
		loc = time.UTC
	}
	if sprint.State == SprintStateFuture || sprint.StartDate == "" || sprint.EndDate == "" {
		return Burndown{}, ErrSprintNotStarted
	}
	startDate, err := time.ParseInLocation(sprintDateLayout, sprint.StartDate, loc)
	if err != nil {
		return Burndown{}, err
	}
	endDate, err := time.ParseInLocation(sprintDateLayout, sprint.EndDate, loc)
	if err != nil {
		return Burndown{}, err
	}
	start := startDate
	if sprint.StartedAt != nil {
		start = *sprint.StartedAt
	}
	until := in.Now
	if sprint.CompletedAt != nil {
		until = *sprint.CompletedAt
	}
	if until.Before(start) {
		until = start
	}
	result := Burndown{SprintID: sprint.ID, Start: start, End: endDate.AddDate(0, 0, 1), Until: until, Events: []BurndownEvent{}, Days: []BurndownDay{}}

	issues := burndownIssues(in)

	// 起点の状態
	for _, issue := range issues {
		issue.exists = !issue.todo.CreatedAt.After(start)
		issue.member = valueAt(issue.steps, valueMember, start, boolValue(issue.member)) == 1
		issue.done = valueAt(issue.steps, valueDone, start, boolValue(issue.todo.Done)) == 1
		issue.points = valueAt(issue.steps, valuePoints, start, issue.points)
		result.StartPoints += issue.remaining()
	}

	// 起点より後の変化を、起きた順に当てる。完了したスプリントでは、完了と同時かそれより後にスプリントから外れた変更
	// （完了したときの未完了の課題の移動）は含めない
	type change struct {
		issue *burndownIssue
		step  burndownStep
		// created は課題の作成（step は使わない）
		created bool
	}
	changes := []change{}
	for _, issue := range issues {
		if created := issue.todo.CreatedAt; created.After(start) && !created.After(until) {
			changes = append(changes, change{issue: issue, step: burndownStep{at: created, order: -1}, created: true})
		}
		for _, step := range issue.steps {
			if !step.at.After(start) || step.at.After(until) {
				continue
			}
			if step.value == valueMember && step.new == 0 && sprint.CompletedAt != nil && !step.at.Before(until) {
				continue
			}
			changes = append(changes, change{issue: issue, step: step})
		}
	}
	sort.SliceStable(changes, func(a, b int) bool {
		if !changes[a].step.at.Equal(changes[b].step.at) {
			return changes[a].step.at.Before(changes[b].step.at)
		}
		return changes[a].step.order < changes[b].step.order
	})

	remaining := result.StartPoints
	for _, c := range changes {
		issue := c.issue
		wasIn, wasDone, before := issue.inSprint(), issue.done, issue.remaining()
		switch {
		case c.created:
			issue.exists = true
		case c.step.value == valueMember:
			issue.member = c.step.new == 1
		case c.step.value == valueDone:
			issue.done = c.step.new == 1
		case c.step.value == valuePoints:
			issue.points = c.step.new
		}
		delta := issue.remaining() - before
		var kind BurndownEventKind
		switch {
		case wasIn != issue.inSprint():
			kind = BurndownRemoved
			if issue.inSprint() {
				kind = BurndownAdded
			}
		case !issue.inSprint():
			continue
		case wasDone != issue.done:
			kind = BurndownReopened
			if issue.done {
				kind = BurndownCompleted
			}
		case delta != 0:
			kind = BurndownEstimate
		default:
			continue
		}
		remaining += delta
		at := c.step.at
		result.Events = append(result.Events, BurndownEvent{
			Time: at, Date: at.In(loc).Format(sprintDateLayout), TodoID: issue.todo.ID, Key: issue.todo.Key, Title: issue.todo.Title,
			Kind: kind, Points: issue.pointsValue(), Change: delta, Remaining: remaining,
		})
	}

	result.Days = burndownDays(result, startDate, endDate, loc)
	return result, nil
}

// burndownIssues は集計に入れる課題（サブタスクを除き、今スプリントに入っているか、記録したか、履歴でスプリントに入ったことがある課題）と、
// その状態の変化を返す。member・points には履歴がないときの値を入れておく
func burndownIssues(in BurndownInput) []*burndownIssue {
	sprint := in.Sprint
	recorded := map[int]bool{}
	for _, r := range in.Recorded {
		recorded[r.TodoID] = true
	}
	byTodo := map[int][]HistoryEntry{}
	for _, entry := range in.History {
		byTodo[entry.TodoID] = append(byTodo[entry.TodoID], entry)
	}

	issues := []*burndownIssue{}
	for _, todo := range in.Todos {
		if todo.ParentID != 0 {
			continue
		}
		entries := byTodo[todo.ID]
		sort.SliceStable(entries, func(a, b int) bool { return entries[a].ID < entries[b].ID })
		touched := false
		issue := &burndownIssue{todo: todo, member: todo.SprintID == sprint.ID || recorded[todo.ID], points: math.NaN()}
		if todo.StoryPoints != nil {
			issue.points = *todo.StoryPoints
		}
		for _, entry := range entries {
			step := burndownStep{at: entry.CreatedAt, order: entry.ID, hasOld: true}
			switch entry.Field {
			case HistoryFieldSprint:
				step.value = valueMember
				step.old = boolValue(refersToSprint(entry.OldRefID, entry.OldValue, sprint))
				step.new = boolValue(refersToSprint(entry.NewRefID, entry.NewValue, sprint))
				touched = touched || step.old == 1 || step.new == 1
			case HistoryFieldSpace:
				// スペースが変わると、課題はバックログに戻る（変更前にスプリントに入っていたかは分からない）
				step.value = valueMember
				step.hasOld = false
			case HistoryFieldStatusCategory:
				step.value = valueDone
				step.old = boolValue(isValue(entry.OldValue, string(StatusCategoryDone)))
				step.new = boolValue(isValue(entry.NewValue, string(StatusCategoryDone)))
			case HistoryFieldDone:
				step.value = valueDone
				step.old = boolValue(isValue(entry.OldValue, "true"))
				step.new = boolValue(isValue(entry.NewValue, "true"))
			case HistoryFieldStoryPoints:
				step.value = valuePoints
				step.old = parsePoints(entry.OldValue)
				step.new = parsePoints(entry.NewValue)
			default:
				continue
			}
			issue.steps = append(issue.steps, step)
		}
		if issue.member || touched {
			issues = append(issues, issue)
		}
	}
	return issues
}

// valueAt は at の時点の値を返す（ポイントの未設定は NaN）。at までの最後の変化の変更後、なければ at より後の最初の変化の変更前、どちらもなければ fallback
func valueAt(steps []burndownStep, value burndownValue, at time.Time, fallback float64) float64 {
	var last *burndownStep
	for i := range steps {
		step := &steps[i]
		if step.value != value {
			continue
		}
		if !step.at.After(at) {
			last = step
			continue
		}
		if last != nil {
			return last.new
		}
		if step.hasOld {
			return step.old
		}
	}
	if last != nil {
		return last.new
	}
	return fallback
}

// burndownDays は開始日から終了日（実績が終了日より後まであればその日）までの、日ごとの増減と残りポイントを返す
func burndownDays(b Burndown, startDate, endDate time.Time, loc *time.Location) []BurndownDay {
	untilDate := b.Until.In(loc).Format(sprintDateLayout)
	last := endDate
	if u, err := time.ParseInLocation(sprintDateLayout, untilDate, loc); err == nil && u.After(last) {
		last = u
	}
	days := []BurndownDay{}
	remaining := b.StartPoints
	next := 0
	for d := startDate; !d.After(last); d = d.AddDate(0, 0, 1) {
		day := BurndownDay{Date: d.Format(sprintDateLayout)}
		for next < len(b.Events) && b.Events[next].Date <= day.Date {
			event := b.Events[next]
			switch event.Kind {
			case BurndownAdded:
				day.Added += event.Change
			case BurndownRemoved:
				day.Removed += event.Change
			case BurndownEstimate:
				day.Estimate += event.Change
			default:
				day.Completed += event.Change
			}
			remaining = event.Remaining
			next++
		}
		if day.Date <= untilDate {
			r := remaining
			day.Remaining = &r
		}
		days = append(days, day)
	}
	return days
}

// refersToSprint は履歴のスプリントの値が sprint を指すか。ID があれば ID で、なければ（ID を残す前の記録）名前で比べる
func refersToSprint(ref *int, value *string, sprint Sprint) bool {
	if ref != nil {
		return *ref == sprint.ID
	}
	return isValue(value, sprint.Name)
}

func isValue(value *string, want string) bool {
	return value != nil && *value == want
}

func boolValue(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

// parsePoints は履歴のストーリーポイントを数値にする。未設定（nil）や読めない値は NaN
func parsePoints(value *string) float64 {
	if value == nil {
		return math.NaN()
	}
	p, err := strconv.ParseFloat(*value, 64)
	if err != nil {
		return math.NaN()
	}
	return p
}
