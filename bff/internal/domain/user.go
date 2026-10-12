package domain

import (
	"errors"
	"hash/crc32"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// displayNameMax は表示名の最大文字数
	displayNameMax = 50
	// emailMax はメールアドレスの最大文字数
	emailMax = 255
	// passwordMin・passwordMax はパスワードの長さ。bcrypt は 72 バイトを超える部分を無視するため、上限をバイト数で決める
	passwordMin = 8
	passwordMax = 72
)

var (
	ErrInvalidDisplayName = errors.New("表示名は空にできず、50 文字以内にしてください")
	ErrInvalidEmail       = errors.New("メールアドレスの形式が正しくありません")
	ErrInvalidPassword    = errors.New("パスワードは 8 文字以上、72 バイト以内にしてください")
)

// User はアプリを利用するユーザー。パスワードのハッシュは API のレスポンスに含めない
type User struct {
	ID           int
	DisplayName  string
	Email        string
	PasswordHash string    `json:"-"` // bcrypt のハッシュ
	AvatarColor  string    // アバターの丸の色（#RRGGBB）
	CreatedAt    time.Time // 作成日時（UTC）
}

// avatarColors はアバターの色の候補（Atlassian Design System のアクセントカラー）を返す
func avatarColors() []string {
	return []string{
		"#0052CC", // 青
		"#00875A", // 緑
		"#FF5630", // 赤
		"#FF991F", // オレンジ
		"#6554C0", // 紫
		"#00B8D9", // 水色
		"#36B37E", // 明るい緑
		"#172B4D", // 紺
	}
}

// AvatarColorFor はメールアドレスから決まるアバターの色を返す。同じメールアドレスなら常に同じ色になる
func AvatarColorFor(email string) string {
	colors := avatarColors()
	return colors[crc32.ChecksumIEEE([]byte(email))%uint32(len(colors))]
}

// NormalizeEmail はメールアドレスの前後の空白を除いて小文字にする。大文字・小文字の違いだけのアドレスは同じユーザーとみなす
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidateSignup はサインアップの入力を確かめる。displayName と email は正規化済み（前後の空白を除いたもの）を渡す
func ValidateSignup(displayName, email, password string) error {
	if displayName == "" || utf8.RuneCountInString(displayName) > displayNameMax {
		return ErrInvalidDisplayName
	}
	if err := validateEmail(email); err != nil {
		return err
	}
	if utf8.RuneCountInString(password) < passwordMin || len(password) > passwordMax {
		return ErrInvalidPassword
	}
	return nil
}

// validateEmail は「名前 <アドレス>」などの形式を受け付けず、アドレスそのものだけを許す
func validateEmail(email string) error {
	if email == "" || utf8.RuneCountInString(email) > emailMax {
		return ErrInvalidEmail
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return ErrInvalidEmail
	}
	return nil
}
