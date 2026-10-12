package domain

import (
	"strings"
	"testing"
)

func TestNormalizeComponentFields(t *testing.T) {
	if got, err := NormalizeComponentName("  フロントエンド "); err != nil || got != "フロントエンド" {
		t.Errorf("NormalizeComponentName = %q, %v", got, err)
	}
	for _, name := range []string{"", "  ", strings.Repeat("a", 101)} {
		if _, err := NormalizeComponentName(name); err != ErrInvalidComponentName {
			t.Errorf("NormalizeComponentName(%q) error = %v", name, err)
		}
	}
	if got, err := NormalizeComponentDescription(" 画面 "); err != nil || got != "画面" {
		t.Errorf("NormalizeComponentDescription = %q, %v", got, err)
	}
	if _, err := NormalizeComponentDescription(strings.Repeat("あ", 1001)); err != ErrInvalidComponentDescription {
		t.Errorf("long description error = %v", err)
	}
}
