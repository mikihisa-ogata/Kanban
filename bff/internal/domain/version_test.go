package domain

import (
	"strings"
	"testing"
)

func TestNormalizeVersionFields(t *testing.T) {
	if got, err := NormalizeVersionName("  v1.0 "); err != nil || got != "v1.0" {
		t.Errorf("NormalizeVersionName = %q, %v", got, err)
	}
	for _, name := range []string{"", "  ", strings.Repeat("a", 101)} {
		if _, err := NormalizeVersionName(name); err != ErrInvalidVersionName {
			t.Errorf("NormalizeVersionName(%q) error = %v", name, err)
		}
	}
	if _, err := NormalizeVersionDescription(strings.Repeat("あ", 1001)); err != ErrInvalidVersionDescription {
		t.Errorf("long description error = %v", err)
	}
	for _, s := range []VersionStatus{VersionUnreleased, VersionReleased, VersionArchived} {
		if !ValidVersionStatus(s) {
			t.Errorf("%s should be valid", s)
		}
	}
	if ValidVersionStatus("done") {
		t.Error("done should be invalid")
	}
}

func TestValidateVersionDates(t *testing.T) {
	tests := []struct {
		start, release string
		ok             bool
	}{
		{"", "", true},
		{"2026-10-01", "", true},
		{"", "2026-11-01", true},
		{"2026-10-01", "2026-11-01", true},
		{"2026-11-01", "2026-10-01", false},
		{"2026/10/01", "", false},
	}
	for _, tt := range tests {
		if err := ValidateVersionDates(tt.start, tt.release); (err == nil) != tt.ok {
			t.Errorf("ValidateVersionDates(%q, %q) = %v", tt.start, tt.release, err)
		}
	}
}
