package domain

import (
	"strings"
	"testing"
)

func TestChangedFields(t *testing.T) {
	entries := []HistoryEntry{
		{Field: HistoryFieldStatus}, {Field: HistoryFieldStatusCategory}, {Field: HistoryFieldAssignee},
		{Field: HistoryFieldSpace}, {Field: HistoryFieldKey}, {Field: HistoryFieldStatus},
	}
	if got := ChangedFields(entries); got != "status,assignee,space" {
		t.Errorf("ChangedFields() = %q", got)
	}
	if got := ChangedFields([]HistoryEntry{{Field: HistoryFieldCreated}}); got != "" {
		t.Errorf("ChangedFields(created) = %q", got)
	}
}

func TestExcerpt(t *testing.T) {
	if got := Excerpt("  1行目\n\n2行目  "); got != "1行目 2行目" {
		t.Errorf("Excerpt() = %q", got)
	}
	long := strings.Repeat("あ", 101)
	if got := Excerpt(long); got != strings.Repeat("あ", 100)+"…" {
		t.Errorf("Excerpt(long) = %q", got)
	}
	if got := Excerpt(strings.Repeat("あ", 100)); got != strings.Repeat("あ", 100) {
		t.Errorf("Excerpt(100) = %q", got)
	}
	if got := Excerpt("@[山田 太郎](user:1) 見てください"); got != "@山田 太郎 見てください" {
		t.Errorf("Excerpt(mention) = %q", got)
	}
}
