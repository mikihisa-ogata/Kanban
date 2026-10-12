package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// Component はスペースのコンポーネント（Jira の機能区分）。課題は Todo.ComponentIDs で複数のコンポーネントに入る
type Component struct {
	ID          int
	SpaceID     int
	Name        string
	Description string // 空文字は説明なし
	// DefaultAssigneeID は既定の担当者（ユーザーの ID）。0 はなし。
	// 担当者が空の課題にこのコンポーネントを付けると、その人が担当者になる
	DefaultAssigneeID int
	// DefaultAssignee は既定の担当者の概要。なしなら nil
	DefaultAssignee *UserSummary
	CreatedAt       time.Time
}

// componentNameMax はコンポーネント名の最大文字数（DB の VARCHAR(100)）
const componentNameMax = 100

// componentDescriptionMax は説明の最大文字数
const componentDescriptionMax = 1000

var ErrInvalidComponentName = errors.New("コンポーネントの名前は 1〜100 文字で入力してください")

var ErrInvalidComponentDescription = errors.New("コンポーネントの説明は 1000 文字以内で入力してください")

// NormalizeComponentName は前後の空白を除いたコンポーネント名を返す。空や長すぎる名前は ErrInvalidComponentName
func NormalizeComponentName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > componentNameMax {
		return "", ErrInvalidComponentName
	}
	return name, nil
}

// NormalizeComponentDescription は前後の空白を除いた説明を返す。長すぎる説明は ErrInvalidComponentDescription
func NormalizeComponentDescription(description string) (string, error) {
	description = strings.TrimSpace(description)
	if utf8.RuneCountInString(description) > componentDescriptionMax {
		return "", ErrInvalidComponentDescription
	}
	return description, nil
}
