package domain

import (
	"reflect"
	"testing"
)

func TestMentionedUserIDs(t *testing.T) {
	text := "@[山田 太郎](user:1) さん、@[佐藤 花子](user:12) さん確認を。@[山田 太郎](user:1) @[x](user:0) [リンク](user:3) @山田"
	if got := MentionedUserIDs(text); !reflect.DeepEqual(got, []int{1, 12}) {
		t.Errorf("MentionedUserIDs() = %v", got)
	}
	if got := MentionedUserIDs("メンションなし"); len(got) != 0 {
		t.Errorf("MentionedUserIDs(none) = %v", got)
	}
}

func TestNewlyMentionedUserIDs(t *testing.T) {
	before := "@[A](user:1) よろしく"
	after := "@[A](user:1) @[B](user:2) @[C](user:3) よろしく"
	if got := NewlyMentionedUserIDs(before, after); !reflect.DeepEqual(got, []int{2, 3}) {
		t.Errorf("NewlyMentionedUserIDs() = %v", got)
	}
	if got := NewlyMentionedUserIDs(after, before); len(got) != 0 {
		t.Errorf("NewlyMentionedUserIDs(removed) = %v", got)
	}
}

func TestPlainMentions(t *testing.T) {
	if got := PlainMentions("@[山田 太郎](user:1) さん、見てください"); got != "@山田 太郎 さん、見てください" {
		t.Errorf("PlainMentions() = %q", got)
	}
}
