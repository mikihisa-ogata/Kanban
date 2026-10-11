package domain

import "testing"

func TestDiffTodo(t *testing.T) {
	names := HistoryNames{Epics: map[int]string{1: "認証"}, Spaces: map[int]string{1: "開発", 2: "運用"}}
	three := 3.0
	half := 0.5
	before := Todo{ID: 7, Title: "a", Status: StatusOpen, IssueType: IssueTypeTask, Priority: PriorityMedium,
		Labels: []string{"ui"}, StoryPoints: &three, EpicID: 1, SpaceID: 1, Key: "DEV-1"}

	if entries := DiffTodo(before, before, names); len(entries) != 0 {
		t.Errorf("no changes: entries = %+v", entries)
	}

	after := before
	after.Title = "b"
	after.Description = "説明"
	after.Status = StatusInProgress
	after.Labels = []string{"api", "ui"}
	after.StoryPoints = &half
	after.EpicID = 0
	after.SpaceID = 2
	after.Key = "OPS-1"
	got := map[string][2]string{}
	for _, e := range DiffTodo(before, after, names) {
		if e.TodoID != 7 {
			t.Errorf("TodoID = %d", e.TodoID)
		}
		got[e.Field] = [2]string{value(e.OldValue), value(e.NewValue)}
	}
	want := map[string][2]string{
		HistoryFieldTitle:       {"a", "b"},
		HistoryFieldDescription: {"<nil>", "説明"},
		HistoryFieldStatus:      {"Open", "InProgress"},
		HistoryFieldLabels:      {"ui", "api, ui"},
		HistoryFieldStoryPoints: {"3", "0.5"},
		HistoryFieldEpic:        {"認証", "<nil>"},
		HistoryFieldSpace:       {"開発", "運用"},
		HistoryFieldKey:         {"DEV-1", "OPS-1"},
	}
	if len(got) != len(want) {
		t.Errorf("fields = %v, want %v", got, want)
	}
	for field, w := range want {
		if got[field] != w {
			t.Errorf("%s = %v, want %v", field, got[field], w)
		}
	}

	// 名前が分からないスペース（削除済みなど）は ID で残す
	moved := before
	moved.SpaceID = 9
	entries := DiffTodo(before, moved, names)
	if len(entries) != 1 || value(entries[0].NewValue) != "ID 9" {
		t.Errorf("unknown space: entries = %+v", entries)
	}
}

func value(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
