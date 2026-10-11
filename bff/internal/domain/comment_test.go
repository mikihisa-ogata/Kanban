package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateCommentBody(t *testing.T) {
	tests := []struct {
		body string
		ok   bool
	}{
		{"コメント", true},
		{"**太字**\n- 箇条書き", true},
		{"", false},
		{" \n\t", false},
		{strings.Repeat("あ", 10000), true},
		{strings.Repeat("あ", 10001), false},
	}
	for _, tt := range tests {
		err := ValidateCommentBody(tt.body)
		if tt.ok && err != nil {
			t.Errorf("ValidateCommentBody(%.20q) = %v, want nil", tt.body, err)
		}
		if !tt.ok && !errors.Is(err, ErrInvalidComment) {
			t.Errorf("ValidateCommentBody(%.20q) = %v, want ErrInvalidComment", tt.body, err)
		}
	}
}
