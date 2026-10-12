package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

// VersionStatus はバージョンの状態（Jira のリリースの UNRELEASED / RELEASED / ARCHIVED）
type VersionStatus string

const (
	VersionUnreleased VersionStatus = "unreleased"
	VersionReleased   VersionStatus = "released"
	VersionArchived   VersionStatus = "archived"
)

// ValidVersionStatus はバージョンの状態として使える値かを返す
func ValidVersionStatus(s VersionStatus) bool {
	return s == VersionUnreleased || s == VersionReleased || s == VersionArchived
}

// Version はスペースのバージョン（Jira のリリース・修正バージョン）。課題は Todo.FixVersionIDs で複数のバージョンに入る
type Version struct {
	ID          int
	SpaceID     int
	Name        string
	Description string // 空文字は説明なし
	Status      VersionStatus
	// StartDate は開始日、ReleaseDate はリリース日（例: 2026-11-01）。空文字は未設定
	StartDate   string
	ReleaseDate string
	CreatedAt   time.Time
}

// versionNameMax はバージョン名の最大文字数（DB の VARCHAR(100)）
const versionNameMax = 100

// versionDescriptionMax は説明の最大文字数
const versionDescriptionMax = 1000

var ErrInvalidVersionName = errors.New("バージョンの名前は 1〜100 文字で入力してください")

var ErrInvalidVersionDescription = errors.New("バージョンの説明は 1000 文字以内で入力してください")

var ErrInvalidVersionDates = errors.New("日付は YYYY-MM-DD 形式で、リリース日を開始日以降にしてください")

var ErrInvalidVersionStatus = errors.New("状態は unreleased・released・archived のいずれかにしてください")

// NormalizeVersionName は前後の空白を除いたバージョン名を返す。空や長すぎる名前は ErrInvalidVersionName
func NormalizeVersionName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > versionNameMax {
		return "", ErrInvalidVersionName
	}
	return name, nil
}

// NormalizeVersionDescription は前後の空白を除いた説明を返す。長すぎる説明は ErrInvalidVersionDescription
func NormalizeVersionDescription(description string) (string, error) {
	description = strings.TrimSpace(description)
	if utf8.RuneCountInString(description) > versionDescriptionMax {
		return "", ErrInvalidVersionDescription
	}
	return description, nil
}

// ValidateVersionDates は開始日・リリース日を検証する。どちらも空にでき、両方あるときはリリース日を開始日以降にする
func ValidateVersionDates(startDate, releaseDate string) error {
	var start, release time.Time
	var err error
	if startDate != "" {
		if start, err = time.Parse(sprintDateLayout, startDate); err != nil {
			return ErrInvalidVersionDates
		}
	}
	if releaseDate != "" {
		if release, err = time.Parse(sprintDateLayout, releaseDate); err != nil {
			return ErrInvalidVersionDates
		}
	}
	if startDate != "" && releaseDate != "" && release.Before(start) {
		return ErrInvalidVersionDates
	}
	return nil
}
