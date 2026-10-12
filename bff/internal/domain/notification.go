package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

// NotificationKind は通知の種類
type NotificationKind string

const (
	// NotificationUpdated はウォッチ中の課題の項目が変わったこと。Detail は変わった項目名（HistoryField*）を「,」でつないだもの
	NotificationUpdated NotificationKind = "updated"
	// NotificationCommented はウォッチ中の課題にコメントが付いたこと。Detail はコメントの冒頭
	NotificationCommented NotificationKind = "commented"
	// NotificationMentioned はコメントか説明でメンションされたこと。Detail はその本文の冒頭
	NotificationMentioned NotificationKind = "mentioned"
)

// excerptMax は通知に載せる本文の冒頭の最大文字数
const excerptMax = 100

// Notification はユーザー（UserID）へのアプリ内通知
type Notification struct {
	ID     int
	UserID int
	TodoID int
	// IssueKey・IssueTitle は通知を見たときの課題キーと要約。API の出力専用
	IssueKey   string
	IssueTitle string
	// ActorID は変更・コメントしたユーザーの ID、Actor はその概要（API の出力専用。いなければ nil）
	ActorID   int
	Actor     *UserSummary
	Kind      NotificationKind
	Detail    string
	Read      bool      // 既読か
	CreatedAt time.Time // 通知した日時（UTC）
}

// ChangedFields は通知に載せる、変わった項目名を重複なしで記録順に「,」でつないで返す。
// 作成とステータスのカテゴリ（ステータスと同時に変わる）、課題キー（スペースと同時に変わる）は載せない。載せる項目がなければ空文字
func ChangedFields(entries []HistoryEntry) string {
	seen := map[string]bool{}
	fields := []string{}
	for _, e := range entries {
		switch e.Field {
		case HistoryFieldCreated, HistoryFieldStatusCategory, HistoryFieldKey:
			continue
		}
		if !seen[e.Field] {
			seen[e.Field] = true
			fields = append(fields, e.Field)
		}
	}
	return strings.Join(fields, ",")
}

// Excerpt は通知に載せるコメント・説明の冒頭を返す。メンションは「@表示名」にし、空白と改行は 1 つの空白にまとめ、長ければ「…」で切る
func Excerpt(body string) string {
	text := strings.Join(strings.Fields(PlainMentions(body)), " ")
	if utf8.RuneCountInString(text) <= excerptMax {
		return text
	}
	return string([]rune(text)[:excerptMax]) + "…"
}
