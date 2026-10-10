package domain

import (
	"regexp"
	"strconv"
	"strings"
)

type Space struct {
	ID    int
	Title string
	Key   string // 課題キーの接頭辞（例: KAN）。全スペースで一意
}

// UnassignedKeyPrefix はスペース未割り当てのタスクの課題キーの接頭辞。スペースのキーには使えない
const UnassignedKeyPrefix = "TASK"

const (
	spaceKeyMinLength = 2
	spaceKeyMaxLength = 10
	// fallbackSpaceKey は名前から英字を取り出せないとき（日本語の名前など）に使うキー
	fallbackSpaceKey = "SP"
)

var spaceKeyPattern = regexp.MustCompile(`^[A-Z][A-Z0-9]{1,9}$`)

// ValidSpaceKey はスペースのキーとして使えるかを返す。
// 英大文字で始まる英大文字・数字の 2〜10 文字で、UnassignedKeyPrefix 以外
func ValidSpaceKey(key string) bool {
	return spaceKeyPattern.MatchString(key) && key != UnassignedKeyPrefix
}

// IssueKey は課題キー（例: KAN-12）を返す。接頭辞か番号がない場合は空文字
func IssueKey(prefix string, number int) string {
	if prefix == "" || number <= 0 {
		return ""
	}
	return prefix + "-" + strconv.Itoa(number)
}

var asciiWordPattern = regexp.MustCompile(`[A-Za-z][A-Za-z0-9]*`)

// SuggestSpaceKey はスペース名からキーの候補を作る。taken に含まれるキーとは重ならないよう、末尾に数字を付ける。
// 英単語が 2 つ以上ならその頭文字、1 つなら先頭の 3 文字を使う（例: "Kanban" → KAN、"Mobile App" → MA）
func SuggestSpaceKey(title string, taken map[string]bool) string {
	words := asciiWordPattern.FindAllString(title, -1)
	base := ""
	if len(words) >= 2 {
		for _, word := range words {
			base += word[:1]
		}
	} else if len(words) == 1 {
		base = words[0]
		if len(base) > 3 {
			base = base[:3]
		}
	}
	base = strings.ToUpper(base)
	if len(base) > spaceKeyMaxLength {
		base = base[:spaceKeyMaxLength]
	}
	if len(base) < spaceKeyMinLength {
		base = fallbackSpaceKey
	}

	if ValidSpaceKey(base) && !taken[base] {
		return base
	}
	for n := 2; ; n++ {
		suffix := strconv.Itoa(n)
		prefix := base
		if len(prefix)+len(suffix) > spaceKeyMaxLength {
			prefix = prefix[:spaceKeyMaxLength-len(suffix)]
		}
		if key := prefix + suffix; ValidSpaceKey(key) && !taken[key] {
			return key
		}
	}
}
