package domain

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeLabels(t *testing.T) {
	tests := []struct {
		name    string
		in      []string
		want    []string
		wantErr bool
	}{
		{"nil は空", nil, []string{}, false},
		{"前後の空白を除いて名前順", []string{" ui ", "Backend", "api"}, []string{"api", "Backend", "ui"}, false},
		{"大文字・小文字だけ違う重複は最初のものを残す", []string{"Bug", "bug", "BUG"}, []string{"Bug"}, false},
		{"日本語も使える", []string{"不具合", "画面"}, []string{"不具合", "画面"}, false},
		{"空は不可", []string{"ui", "  "}, nil, true},
		{"途中の空白は不可", []string{"two words"}, nil, true},
		{"全角空白も不可", []string{"二つ　の"}, nil, true},
		{"255 文字まで", []string{strings.Repeat("あ", 255)}, []string{strings.Repeat("あ", 255)}, false},
		{"256 文字は不可", []string{strings.Repeat("あ", 256)}, nil, true},
	}
	for _, tt := range tests {
		got, err := NormalizeLabels(tt.in)
		if tt.wantErr {
			if !errors.Is(err, ErrInvalidLabel) {
				t.Errorf("%s: err = %v, want ErrInvalidLabel", tt.name, err)
			}
			continue
		}
		if err != nil || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: NormalizeLabels(%q) = %q, %v, want %q", tt.name, tt.in, got, err, tt.want)
		}
	}
}
