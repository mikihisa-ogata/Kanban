package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeFilterName(t *testing.T) {
	tests := []struct {
		name    string
		want    string
		wantErr error
	}{
		{" 自分のバグ ", "自分のバグ", nil},
		{"", "", ErrInvalidFilterName},
		{"   ", "", ErrInvalidFilterName},
		{strings.Repeat("あ", 100), strings.Repeat("あ", 100), nil},
		{strings.Repeat("あ", 101), "", ErrInvalidFilterName},
	}
	for _, tt := range tests {
		got, err := NormalizeFilterName(tt.name)
		if !errors.Is(err, tt.wantErr) || got != tt.want {
			t.Errorf("NormalizeFilterName(%q) = %q, %v; want %q, %v", tt.name, got, err, tt.want, tt.wantErr)
		}
	}
}

func TestNormalizeFilterJQL(t *testing.T) {
	if got, err := NormalizeFilterJQL("  project = DEV  "); err != nil || got != "project = DEV" {
		t.Errorf("NormalizeFilterJQL = %q, %v", got, err)
	}
	if got, err := NormalizeFilterJQL(""); err != nil || got != "" {
		t.Errorf("NormalizeFilterJQL(empty) = %q, %v", got, err)
	}
	if _, err := NormalizeFilterJQL(strings.Repeat("a", 4001)); !errors.Is(err, ErrInvalidFilterJQL) {
		t.Errorf("NormalizeFilterJQL(too long) err = %v", err)
	}
}
