package domain

import "testing"

func TestValidSpaceKey(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"KAN", true},
		{"AB", true},
		{"A1", true},
		{"ABCDEFGHIJ", true},
		{"A", false},
		{"ABCDEFGHIJK", false},
		{"kan", false},
		{"1AB", false},
		{"KA-N", false},
		{"", false},
		{UnassignedKeyPrefix, false},
	}
	for _, tt := range tests {
		if got := ValidSpaceKey(tt.key); got != tt.want {
			t.Errorf("ValidSpaceKey(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}

func TestIssueKey(t *testing.T) {
	if got := IssueKey("KAN", 12); got != "KAN-12" {
		t.Errorf("IssueKey(KAN, 12) = %q", got)
	}
	if got := IssueKey("", 12); got != "" {
		t.Errorf("IssueKey(\"\", 12) = %q, want empty", got)
	}
	if got := IssueKey("KAN", 0); got != "" {
		t.Errorf("IssueKey(KAN, 0) = %q, want empty", got)
	}
}

func TestParseIssueKey(t *testing.T) {
	tests := []struct {
		key        string
		wantPrefix string
		wantNumber int
		wantOK     bool
	}{
		{"KAN-12", "KAN", 12, true},
		{"kan-12", "KAN", 12, true},
		{"TASK-3", "TASK", 3, true},
		{"A1B-1", "A1B", 1, true},
		{"KAN-0", "", 0, false},
		{"KAN-012", "", 0, false},
		{"KAN-+1", "", 0, false},
		{"KAN12", "", 0, false},
		{"KAN-", "", 0, false},
		{"-12", "", 0, false},
		{"1KAN-12", "", 0, false},
		{"KAN-1-2", "", 0, false},
		{"KAN-9999999999", "", 0, false},
		{"", "", 0, false},
	}
	for _, tt := range tests {
		prefix, number, ok := ParseIssueKey(tt.key)
		if prefix != tt.wantPrefix || number != tt.wantNumber || ok != tt.wantOK {
			t.Errorf("ParseIssueKey(%q) = (%q, %d, %v), want (%q, %d, %v)",
				tt.key, prefix, number, ok, tt.wantPrefix, tt.wantNumber, tt.wantOK)
		}
	}
}

func TestSuggestSpaceKey(t *testing.T) {
	tests := []struct {
		name  string
		title string
		taken []string
		want  string
	}{
		{"1 単語は先頭 3 文字", "Kanban", nil, "KAN"},
		{"短い単語はそのまま", "Go", nil, "GO"},
		{"2 単語以上は頭文字", "Mobile App", nil, "MA"},
		{"記号や日本語は区切りとして扱う", "社内 web-site リニューアル", nil, "WS"},
		{"頭文字は 10 文字まで", "a b c d e f g h i j k l", nil, "ABCDEFGHIJ"},
		{"英字がなければ SP", "日本おすすめ地図", nil, "SP"},
		{"1 文字しか取れなければ SP", "A案", nil, "SP"},
		{"重複したら数字を付ける", "お金", []string{"SP"}, "SP2"},
		{"数字も重複したら次の数字", "Kanban", []string{"KAN", "KAN2"}, "KAN3"},
		{"10 文字を超えないよう削る", "a b c d e f g h i j", []string{"ABCDEFGHIJ"}, "ABCDEFGHI2"},
		{"予約語は使わない", "Task", nil, "TAS"},
		{"予約語そのものになる場合は数字を付ける", "T A S K", nil, "TASK2"},
	}
	for _, tt := range tests {
		taken := map[string]bool{}
		for _, key := range tt.taken {
			taken[key] = true
		}
		got := SuggestSpaceKey(tt.title, taken)
		if got != tt.want {
			t.Errorf("%s: SuggestSpaceKey(%q) = %q, want %q", tt.name, tt.title, got, tt.want)
		}
		if !ValidSpaceKey(got) {
			t.Errorf("%s: SuggestSpaceKey(%q) = %q is not a valid key", tt.name, tt.title, got)
		}
	}
}
