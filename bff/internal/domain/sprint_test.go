package domain

import (
	"strings"
	"testing"
)

func TestNormalizeSprintName(t *testing.T) {
	if got, err := NormalizeSprintName("  DEV スプリント 1 "); err != nil || got != "DEV スプリント 1" {
		t.Errorf("NormalizeSprintName = %q, %v", got, err)
	}
	for _, name := range []string{"", "   ", strings.Repeat("あ", 101)} {
		if _, err := NormalizeSprintName(name); err != ErrInvalidSprintName {
			t.Errorf("NormalizeSprintName(%q) error = %v", name, err)
		}
	}
	if _, err := NormalizeSprintGoal(strings.Repeat("あ", 1001)); err != ErrInvalidSprintGoal {
		t.Errorf("long goal error = %v", err)
	}
}

func TestValidateSprintDates(t *testing.T) {
	tests := []struct {
		start, end string
		required   bool
		ok         bool
	}{
		{"", "", false, true},
		{"", "", true, false},
		{"2026-10-12", "2026-10-26", true, true},
		{"2026-10-12", "2026-10-12", true, true},
		{"2026-10-12", "2026-10-11", false, false},
		{"2026-10-12", "", false, false},
		{"2026/10/12", "2026-10-26", false, false},
	}
	for _, tt := range tests {
		err := ValidateSprintDates(tt.start, tt.end, tt.required)
		if (err == nil) != tt.ok {
			t.Errorf("ValidateSprintDates(%q, %q, %v) = %v", tt.start, tt.end, tt.required, err)
		}
	}
}
