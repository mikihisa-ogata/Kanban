package domain

import (
	"strings"
	"testing"
)

func TestValidateSignup(t *testing.T) {
	tests := []struct {
		name        string
		displayName string
		email       string
		password    string
		want        error
	}{
		{"ok", "山田 太郎", "taro@example.com", "password", nil},
		{"表示名が空", "", "taro@example.com", "password", ErrInvalidDisplayName},
		{"表示名が 51 文字", strings.Repeat("あ", 51), "taro@example.com", "password", ErrInvalidDisplayName},
		{"表示名が 50 文字", strings.Repeat("あ", 50), "taro@example.com", "password", nil},
		{"メールが空", "taro", "", "password", ErrInvalidEmail},
		{"@ がない", "taro", "taro.example.com", "password", ErrInvalidEmail},
		{"名前付きの形式", "taro", "Taro <taro@example.com>", "password", ErrInvalidEmail},
		{"パスワードが 7 文字", "taro", "taro@example.com", "1234567", ErrInvalidPassword},
		{"パスワードが 72 バイト", "taro", "taro@example.com", strings.Repeat("a", 72), nil},
		{"パスワードが 73 バイト", "taro", "taro@example.com", strings.Repeat("a", 73), ErrInvalidPassword},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateSignup(tt.displayName, tt.email, tt.password); err != tt.want {
				t.Errorf("ValidateSignup() = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  Taro@Example.COM "); got != "taro@example.com" {
		t.Errorf("NormalizeEmail() = %q", got)
	}
}

func TestAvatarColorFor(t *testing.T) {
	a := AvatarColorFor("taro@example.com")
	if a != AvatarColorFor("taro@example.com") {
		t.Error("同じメールアドレスで色が変わる")
	}
	if len(a) != 7 || a[0] != '#' {
		t.Errorf("AvatarColorFor() = %q", a)
	}
}
