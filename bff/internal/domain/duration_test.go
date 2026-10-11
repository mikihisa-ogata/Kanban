package domain

import (
	"errors"
	"testing"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		in   string
		want int
	}{
		{"30m", 30},
		{"3h", 180},
		{"1d", 480},
		{"1w", 2400},
		{"1w 2d 3h 30m", 2400 + 960 + 180 + 30},
		{"3h30m", 210},
		{"  2D  4H ", 960 + 240},
		{"30m 1h", 90},
		{"1h 1h", 120},
		{"1.5h", 90},
		{"0.5d", 240},
		{"0m", 0},
		{"0.01m", 0},
	}
	for _, tt := range tests {
		got, err := ParseDuration(tt.in)
		if err != nil || got != tt.want {
			t.Errorf("ParseDuration(%q) = %d, %v, want %d", tt.in, got, err, tt.want)
		}
	}

	for _, in := range []string{"", "  ", "3", "h", "3x", "3 h", "-1h", "1h abc", "1,5h", "99999w"} {
		if got, err := ParseDuration(in); !errors.Is(err, ErrInvalidDuration) {
			t.Errorf("ParseDuration(%q) = %d, %v, want ErrInvalidDuration", in, got, err)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := map[int]string{
		0:                     "0m",
		-5:                    "0m",
		30:                    "30m",
		60:                    "1h",
		90:                    "1h 30m",
		480:                   "1d",
		2400:                  "1w",
		2400 + 960 + 180 + 30: "1w 2d 3h 30m",
		2*2400 + 480 + 1:      "2w 1d 1m",
	}
	for in, want := range tests {
		if got := FormatDuration(in); got != want {
			t.Errorf("FormatDuration(%d) = %q, want %q", in, got, want)
		}
	}
	// 表示した文字列を読み直すと同じ分になる
	for _, minutes := range []int{1, 59, 61, 479, 481, 2399, 2401, 12345} {
		if got, err := ParseDuration(FormatDuration(minutes)); err != nil || got != minutes {
			t.Errorf("round trip %d -> %q -> %d, %v", minutes, FormatDuration(minutes), got, err)
		}
	}
}
