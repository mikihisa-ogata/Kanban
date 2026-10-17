package domain

import "testing"

func TestValidateEpicDates(t *testing.T) {
	tests := []struct {
		start, due string
		ok         bool
	}{
		{"", "", true},
		{"2026-10-01", "", true},
		{"", "2026-11-01", true},
		{"2026-10-01", "2026-10-01", true},
		{"2026-10-01", "2026-11-01", true},
		{"2026-11-01", "2026-10-01", false},
		{"2026/10/01", "", false},
		{"", "2026-02-30", false},
	}
	for _, tt := range tests {
		err := ValidateEpicDates(tt.start, tt.due)
		if (err == nil) != tt.ok {
			t.Errorf("ValidateEpicDates(%q, %q) error = %v", tt.start, tt.due, err)
		}
		if err != nil && err != ErrInvalidEpicDates {
			t.Errorf("ValidateEpicDates(%q, %q) error = %v, want ErrInvalidEpicDates", tt.start, tt.due, err)
		}
	}
}

func TestNormalizeEpicColor(t *testing.T) {
	for in, want := range map[string]string{"": "", " #6554c0 ": "#6554C0", "#00875A": "#00875A"} {
		if got, err := NormalizeEpicColor(in); err != nil || got != want {
			t.Errorf("NormalizeEpicColor(%q) = %q, %v", in, got, err)
		}
	}
	for _, in := range []string{"red", "#123", "#12345G", "6554C0"} {
		if _, err := NormalizeEpicColor(in); err != ErrInvalidEpicColor {
			t.Errorf("NormalizeEpicColor(%q) error = %v", in, err)
		}
	}
}
