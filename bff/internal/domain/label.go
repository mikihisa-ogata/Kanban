package domain

import (
	"errors"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

// labelMaxLength はラベル名の最大の文字数（DB の labels.name に合わせる）
const labelMaxLength = 255

var ErrInvalidLabel = errors.New("ラベルは空白を含まない 255 文字以内にしてください")

// NormalizeLabels はラベル名の前後の空白を除き、重複（大文字・小文字の違いだけのものを含む）を除いて名前順に並べる。
// Jira と同じく空白を含むラベルや空のラベルは使えず、ErrInvalidLabel を返す
func NormalizeLabels(names []string) ([]string, error) {
	labels := []string{}
	seen := map[string]bool{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || utf8.RuneCountInString(name) > labelMaxLength || strings.IndexFunc(name, unicode.IsSpace) >= 0 {
			return nil, ErrInvalidLabel
		}
		folded := strings.ToLower(name)
		if seen[folded] {
			continue
		}
		seen[folded] = true
		labels = append(labels, name)
	}
	sort.Slice(labels, func(i, j int) bool {
		return strings.ToLower(labels[i]) < strings.ToLower(labels[j])
	})
	return labels, nil
}
