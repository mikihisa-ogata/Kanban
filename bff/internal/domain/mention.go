package domain

import (
	"regexp"
	"strconv"
)

// メンションは Markdown の本文に「@[表示名](user:ユーザー ID)」の形で書く（例: @[山田 太郎](user:1)）。
// 表示名に空白があっても誰へのメンションかが ID で決まるように、表示名と ID の両方を持たせる

// mentionPattern はメンションに一致する正規表現を返す。1 つ目のグループが表示名、2 つ目がユーザー ID
func mentionPattern() *regexp.Regexp {
	return regexp.MustCompile(`@\[([^\]\n]+)\]\(user:(\d+)\)`)
}

// MentionedUserIDs は本文でメンションされたユーザーの ID を、重複なしで出てきた順に返す
func MentionedUserIDs(text string) []int {
	ids := []int{}
	seen := map[int]bool{}
	for _, m := range mentionPattern().FindAllStringSubmatch(text, -1) {
		id, err := strconv.Atoi(m[2])
		if err != nil || id == 0 || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
}

// NewlyMentionedUserIDs は after でメンションされていて、before ではメンションされていなかったユーザーの ID を返す。
// 編集で同じ人に何度も通知しないために使う
func NewlyMentionedUserIDs(before, after string) []int {
	old := map[int]bool{}
	for _, id := range MentionedUserIDs(before) {
		old[id] = true
	}
	ids := []int{}
	for _, id := range MentionedUserIDs(after) {
		if !old[id] {
			ids = append(ids, id)
		}
	}
	return ids
}

// PlainMentions はメンションを「@表示名」にした本文を返す（通知に載せる冒頭などに使う）
func PlainMentions(text string) string {
	return mentionPattern().ReplaceAllString(text, "@$1")
}
