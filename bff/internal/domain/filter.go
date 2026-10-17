package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// SavedFilter は名前を付けて保存した検索条件（Jira の保存済みフィルター）。作ったユーザーだけが使う
type SavedFilter struct {
	ID      int
	OwnerID int
	Name    string
	// JQL は課題一覧の検索条件（ORDER BY を含む）。空文字はすべての課題
	JQL string
	// Favourite はお気に入り（スター）。お気に入りのフィルターは左ナビに並ぶ
	Favourite bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// filterNameMax はフィルター名の最大文字数（DB の VARCHAR(100)）
const filterNameMax = 100

// filterJQLMax は保存できる JQL の最大文字数
const filterJQLMax = 4000

var ErrInvalidFilterName = errors.New("フィルターの名前は 1〜100 文字で入力してください")

var ErrInvalidFilterJQL = errors.New("フィルターの JQL は 4000 文字以内で入力してください")

// NormalizeFilterName は前後の空白を除いたフィルター名を返す。空や長すぎる名前は ErrInvalidFilterName
func NormalizeFilterName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > filterNameMax {
		return "", ErrInvalidFilterName
	}
	return name, nil
}

// NormalizeFilterJQL は前後の空白を除いた JQL を返す。長すぎる JQL は ErrInvalidFilterJQL
func NormalizeFilterJQL(jql string) (string, error) {
	jql = strings.TrimSpace(jql)
	if utf8.RuneCountInString(jql) > filterJQLMax {
		return "", ErrInvalidFilterJQL
	}
	return jql, nil
}
