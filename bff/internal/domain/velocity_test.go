package domain

import (
	"testing"
	"time"
)

// closedSprint は day 日（日本時間）に開始し、day+4 日の 18 時に完了したスプリント
func closedSprint(id, day int) Sprint {
	startedAt, completedAt := jst(day, 10), jst(day+4, 18)
	date := func(d int) string { return time.Date(2026, 10, d, 0, 0, 0, 0, tokyo).Format(sprintDateLayout) }
	return Sprint{ID: id, Name: "スプリント " + string(rune('0'+id)), State: SprintStateClosed, StartDate: date(day), EndDate: date(day + 4), StartedAt: &startedAt, CompletedAt: &completedAt}
}

func TestComputeVelocity(t *testing.T) {
	s1, s2 := closedSprint(1, 5), closedSprint(2, 12)
	active := Sprint{ID: 3, State: SprintStateActive, StartDate: "2026-10-19", EndDate: "2026-10-23"}
	name2 := str(s2.Name)

	subtask := issue(6, 0, pts(5), true)
	subtask.ParentID = 3
	todos := []Todo{
		issue(1, 1, pts(5), true),  // スプリント 1 で完了
		issue(2, 2, pts(3), true),  // スプリント 1 で未完了 → スプリント 2 へ。2 で完了
		issue(3, 2, pts(8), false), // スプリント 2 の開始時は 2 pt、途中で 8 pt。未完了でバックログへ
		issue(4, 2, pts(1), true),  // スプリント 2 の途中で追加して完了（コミットメントに入らない）
		issue(5, 2, nil, true),     // ポイント未設定
		subtask,
		issue(7, 3, pts(2), false), // 開始中のスプリントは並べない
	}
	recorded := map[int][]SprintIssue{
		1: {{SprintID: 1, TodoID: 1, StoryPoints: pts(5), Completed: true}, {SprintID: 1, TodoID: 2, StoryPoints: pts(3)}},
		2: {
			{SprintID: 2, TodoID: 2, StoryPoints: pts(3), Completed: true},
			{SprintID: 2, TodoID: 3, StoryPoints: pts(8)},
			{SprintID: 2, TodoID: 4, StoryPoints: pts(1), Completed: true},
			{SprintID: 2, TodoID: 5, Completed: true},
		},
	}
	history := []HistoryEntry{
		sprintEntry(1, 1, jst(4, 9), nil, 0, str(s1.Name), 1),
		sprintEntry(2, 2, jst(4, 9), nil, 0, str(s1.Name), 1),
		sprintEntry(3, 2, jst(9, 18), str(s1.Name), 1, name2, 2), // スプリント 1 の完了で移動
		sprintEntry(4, 3, jst(11, 9), nil, 0, name2, 2),
		entry(5, 3, HistoryFieldStoryPoints, jst(13, 9), str("2"), str("8")),
		sprintEntry(6, 4, jst(13, 10), nil, 0, name2, 2),
		sprintEntry(7, 5, jst(11, 9), nil, 0, name2, 2),
		sprintEntry(8, 3, jst(16, 18), name2, 2, nil, 0), // スプリント 2 の完了でバックログへ
	}

	// 並び順は入力の順によらず、完了した順にする
	got, err := ComputeVelocity(VelocityInput{Sprints: []Sprint{active, s2, s1}, Todos: todos, Recorded: recorded, History: history, Location: tokyo})
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		id                    int
		commitment, completed float64
	}{
		{1, 8, 5}, // 開始時: 1 の 5 + 2 の 3。完了: 1 の 5
		{2, 5, 4}, // 開始時: 2 の 3 + 3 の 2 + 5（未設定）。完了: 2 の 3 + 4 の 1
	}
	if len(got.Sprints) != len(want) {
		t.Fatalf("sprints = %+v", got.Sprints)
	}
	for i, w := range want {
		if g := got.Sprints[i]; g.SprintID != w.id || g.Commitment != w.commitment || g.Completed != w.completed {
			t.Errorf("sprint %d = %+v, want %+v", i, g, w)
		}
	}
	if got.Average != 4.5 {
		t.Errorf("Average = %v, want 4.5", got.Average)
	}
	if got.Sprints[0].Name != s1.Name || got.Sprints[0].StartDate != "2026-10-05" || got.Sprints[0].CompletedAt == nil {
		t.Errorf("sprint fields = %+v", got.Sprints[0])
	}
}

// 直近 7 つだけを古い順に並べ、平均もその 7 つで出す。小数の誤差は丸める
func TestComputeVelocity_LatestSeven(t *testing.T) {
	sprints := []Sprint{}
	recorded := map[int][]SprintIssue{}
	for id := 1; id <= 9; id++ {
		sprints = append(sprints, closedSprint(id, id))
		recorded[id] = []SprintIssue{{SprintID: id, TodoID: id, StoryPoints: pts(0.1), Completed: true}, {SprintID: id, TodoID: 100 + id, StoryPoints: pts(0.2), Completed: true}}
	}
	got, err := ComputeVelocity(VelocityInput{Sprints: sprints, Recorded: recorded, Location: tokyo})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Sprints) != VelocitySprintCount || got.Sprints[0].SprintID != 3 || got.Sprints[6].SprintID != 9 {
		t.Fatalf("sprints = %+v", got.Sprints)
	}
	if got.Sprints[0].Completed != 0.3 || got.Average != 0.3 {
		t.Errorf("Completed = %v, Average = %v, want 0.3", got.Sprints[0].Completed, got.Average)
	}
}

// 完了したスプリントがなければ空で、平均は 0
func TestComputeVelocity_Empty(t *testing.T) {
	got, err := ComputeVelocity(VelocityInput{Sprints: []Sprint{{ID: 1, State: SprintStateFuture}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Sprints) != 0 || got.Sprints == nil || got.Average != 0 {
		t.Errorf("velocity = %+v", got)
	}
}
