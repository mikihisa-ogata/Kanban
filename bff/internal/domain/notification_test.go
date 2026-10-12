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

func TestCommentExcerpt(t *testing.T) {
	if got := CommentExcerpt("  1行目\n\n2行目  "); got != "1行目 2行目" {
		t.Errorf("CommentExcerpt() = %q", got)
	}
	long := strings.Repeat("あ", 101)
	if got := CommentExcerpt(long); got != strings.Repeat("あ", 100)+"…" {
		t.Errorf("CommentExcerpt(long) = %q", got)
	}
	if got := CommentExcerpt(strings.Repeat("あ", 100)); got != strings.Repeat("あ", 100) {
		t.Errorf("CommentExcerpt(100) = %q", got)
	}
}
