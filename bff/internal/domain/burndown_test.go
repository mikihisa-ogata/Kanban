package domain

import (
	"testing"
	"time"
)

var tokyo = time.FixedZone("JST", 9*60*60)

// jst は 2026 年 10 月 day 日 hour 時（日本時間）の UTC の日時を返す
func jst(day, hour int) time.Time {
	return time.Date(2026, 10, day, hour, 0, 0, 0, tokyo).UTC()
}

func pts(p float64) *float64 { return &p }

func str(s string) *string { return &s }

func ref(id int) *int { return &id }

// sprintEntry はスプリントの変更の履歴（ID を残した記録）
func sprintEntry(id, todoID int, at time.Time, oldName *string, oldID int, newName *string, newID int) HistoryEntry {
	return HistoryEntry{ID: id, TodoID: todoID, Field: HistoryFieldSprint, OldValue: oldName, NewValue: newName, OldRefID: ref(oldID), NewRefID: ref(newID), CreatedAt: at}
}

func entry(id, todoID int, field string, at time.Time, oldValue, newValue *string) HistoryEntry {
	return HistoryEntry{ID: id, TodoID: todoID, Field: field, OldValue: oldValue, NewValue: newValue, CreatedAt: at}
}

func issue(id int, sprintID int, points *float64, done bool) Todo {
	return Todo{ID: id, Key: "DEV-" + string(rune('0'+id)), SprintID: sprintID, StoryPoints: points, Done: done, CreatedAt: jst(1, 9)}
}

type wantEvent struct {
	todoID    int
	kind      BurndownEventKind
	change    float64
	remaining float64
}

func checkEvents(t *testing.T, got []BurndownEvent, want []wantEvent) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("events = %+v, want %+v", got, want)
	}
	for i, w := range want {
		g := got[i]
		if g.TodoID != w.todoID || g.Kind != w.kind || g.Change != w.change || g.Remaining != w.remaining {
			t.Errorf("event %d = {todo %d %s %v → %v}, want %+v", i, g.TodoID, g.Kind, g.Change, g.Remaining, w)
		}
	}
}

// 開始中のスプリント: 途中の追加・除外・完了・再開・ポイント変更・作成・スペースの移動を再計算する
func TestComputeBurndown_ActiveSprint(t *testing.T) {
	startedAt := jst(12, 10)
	sprint := Sprint{ID: 1, Name: "スプリント 1", State: SprintStateActive, StartDate: "2026-10-12", EndDate: "2026-10-16", StartedAt: &startedAt}
	name := str("スプリント 1")

	created := issue(5, 1, pts(1), false) // E: スプリント中に作成し、あとから入れた（名前だけの古い記録）
	created.CreatedAt = jst(14, 12)
	subtask := issue(8, 1, pts(5), false) // サブタスクは数えない
	subtask.ParentID = 4
	todos := []Todo{
		issue(1, 1, pts(5), true),  // A: 開始時から入っていて、13 日に完了
		issue(2, 1, pts(3), false), // B: 13 日に追加
		issue(3, 0, pts(2), false), // C: 開始時から入っていて、14 日に除外
		issue(4, 1, pts(8), false), // D: 14 日に 3 → 8 pt
		created,
		issue(6, 1, pts(1), false), // F: 履歴なし（最初から入っていたとみなす）
		issue(7, 0, pts(2), false), // G: 開始前に入れ、15 日に別のスペースへ移動
		subtask,
		issue(9, 1, pts(4), false),  // I: 開始前に完了。完了中のポイント変更は数えず、15 日に再開
		issue(10, 2, pts(5), false), // 別のスプリントの課題
	}
	history := []HistoryEntry{
		sprintEntry(1, 7, jst(10, 9), nil, 0, name, 1),
		entry(2, 9, HistoryFieldStatusCategory, jst(11, 9), str("ToDo"), str("Done")),
		sprintEntry(3, 2, jst(13, 11), nil, 0, name, 1),
		entry(4, 1, HistoryFieldStatusCategory, jst(13, 15), str("InProgress"), str("Done")),
		entry(5, 9, HistoryFieldStoryPoints, jst(13, 16), str("2"), str("4")),
		sprintEntry(6, 3, jst(14, 9), name, 1, nil, 0),
		entry(7, 4, HistoryFieldStoryPoints, jst(14, 10), str("3"), str("8")),
		entry(8, 4, HistoryFieldTitle, jst(14, 10), str("x"), str("y")), // 関係のない項目は無視する
		{ID: 9, TodoID: 5, Field: HistoryFieldSprint, NewValue: name, CreatedAt: jst(14, 12).Add(5 * time.Minute)},
		entry(10, 5, HistoryFieldStoryPoints, jst(14, 12).Add(10*time.Minute), nil, str("1")),
		entry(11, 7, HistoryFieldSpace, jst(15, 9), str("開発"), str("運用")),
		entry(12, 9, HistoryFieldStatusCategory, jst(15, 10), str("Done"), str("InProgress")),
		entry(13, 2, HistoryFieldStatus, jst(15, 11), str("To Do"), str("進行中")),
	}

	got, err := ComputeBurndown(BurndownInput{Sprint: sprint, Todos: todos, History: history, Now: jst(15, 12), Location: tokyo})
	if err != nil {
		t.Fatal(err)
	}
	// 開始時: A 5 + C 2 + D 3 + F 1 + G 2（I は完了済み、B・E はまだ入っていない）
	if got.StartPoints != 13 {
		t.Errorf("StartPoints = %v, want 13", got.StartPoints)
	}
	if !got.Start.Equal(startedAt) || !got.End.Equal(jst(17, 0)) || !got.Until.Equal(jst(15, 12)) {
		t.Errorf("Start/End/Until = %v / %v / %v", got.Start, got.End, got.Until)
	}
	checkEvents(t, got.Events, []wantEvent{
		{2, BurndownAdded, 3, 16},
		{1, BurndownCompleted, -5, 11},
		{3, BurndownRemoved, -2, 9},
		{4, BurndownEstimate, 5, 14},
		{5, BurndownAdded, 0, 14}, // ポイント未設定で追加
		{5, BurndownEstimate, 1, 15},
		{7, BurndownRemoved, -2, 13},
		{9, BurndownReopened, 4, 17},
	})
	if e := got.Events[4]; e.Points != nil || e.Date != "2026-10-14" {
		t.Errorf("added unestimated = %+v", e)
	}
	if e := got.Events[3]; e.Points == nil || *e.Points != 8 || e.Key != "DEV-4" {
		t.Errorf("estimate = %+v", e)
	}

	// 日ごと（16 日はまだ来ていない）
	want := []struct {
		date                                string
		added, removed, estimate, completed float64
		remaining                           *float64
	}{
		{"2026-10-12", 0, 0, 0, 0, pts(13)},
		{"2026-10-13", 3, 0, 0, -5, pts(11)},
		{"2026-10-14", 0, -2, 6, 0, pts(15)},
		{"2026-10-15", 0, -2, 0, 4, pts(17)},
		{"2026-10-16", 0, 0, 0, 0, nil},
	}
	if len(got.Days) != len(want) {
		t.Fatalf("days = %+v", got.Days)
	}
	for i, w := range want {
		d := got.Days[i]
		if d.Date != w.date || d.Added != w.added || d.Removed != w.removed || d.Estimate != w.estimate || d.Completed != w.completed ||
			(d.Remaining == nil) != (w.remaining == nil) || (d.Remaining != nil && *d.Remaining != *w.remaining) {
			t.Errorf("day %d = %+v (remaining %v), want %+v", i, d, d.Remaining, w)
		}
	}
}

// 完了したスプリント: 完了と同時の未完了の課題の移動と、完了より後の変更は数えない。
// 開始日時の記録がなければ開始日の 0 時を起点にし、記録した課題は今は別のスプリントでも最初から入っていたとみなす
func TestComputeBurndown_ClosedSprint(t *testing.T) {
	completedAt := jst(16, 18)
	sprint := Sprint{ID: 1, Name: "スプリント 1", State: SprintStateClosed, StartDate: "2026-10-12", EndDate: "2026-10-16", CompletedAt: &completedAt}
	name, next := str("スプリント 1"), str("スプリント 2")
	todos := []Todo{
		issue(1, 1, pts(3), true),  // 16 日 17 時に完了
		issue(2, 2, pts(5), false), // 完了したときに次のスプリントへ移した
		issue(3, 2, pts(2), false), // 移した記録がない（記録した課題）
		issue(4, 1, pts(1), false), // 完了と同じ秒に追加した（追加は数える）
	}
	recorded := []SprintIssue{{SprintID: 1, TodoID: 1}, {SprintID: 1, TodoID: 2}, {SprintID: 1, TodoID: 3}}
	history := []HistoryEntry{
		sprintEntry(1, 1, jst(12, 5), nil, 0, name, 1), // 開始日の 0 時より後に入れたので追加
		entry(2, 1, HistoryFieldStatusCategory, jst(16, 17), str("ToDo"), str("Done")),
		sprintEntry(3, 2, completedAt, name, 1, next, 2),
		sprintEntry(4, 4, completedAt, nil, 0, name, 1),
		entry(5, 3, HistoryFieldStoryPoints, jst(17, 9), str("1"), str("2")),
	}
	got, err := ComputeBurndown(BurndownInput{Sprint: sprint, Todos: todos, Recorded: recorded, History: history, Now: jst(20, 9), Location: tokyo})
	if err != nil {
		t.Fatal(err)
	}
	// 開始時: 2 の 5 + 3 の 1（17 日に変わる前の値）
	if got.StartPoints != 6 || !got.Start.Equal(jst(12, 0)) || !got.Until.Equal(completedAt) {
		t.Errorf("StartPoints = %v, Start = %v, Until = %v", got.StartPoints, got.Start, got.Until)
	}
	checkEvents(t, got.Events, []wantEvent{
		{1, BurndownAdded, 3, 9},
		{1, BurndownCompleted, -3, 6},
		{4, BurndownAdded, 1, 7},
	})
	if len(got.Days) != 5 || got.Days[4].Remaining == nil || *got.Days[4].Remaining != 7 {
		t.Errorf("days = %+v", got.Days)
	}
}

// 未開始のスプリントは集計できない
func TestComputeBurndown_NotStarted(t *testing.T) {
	_, err := ComputeBurndown(BurndownInput{Sprint: Sprint{ID: 1, State: SprintStateFuture, StartDate: "2026-10-12", EndDate: "2026-10-16"}, Now: jst(12, 0)})
	if err != ErrSprintNotStarted {
		t.Errorf("err = %v, want ErrSprintNotStarted", err)
	}
}
