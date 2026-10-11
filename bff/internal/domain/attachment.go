package domain

import (
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// AttachmentMaxSize は添付ファイル 1 つの上限サイズ（10MB）
const AttachmentMaxSize = 10 << 20

// attachmentNameMax はファイル名の最大文字数（DB の VARCHAR(255) に収まる長さ）
const attachmentNameMax = 255

// Attachment は課題の添付ファイルの情報。ファイル本体は Repository が保存先のディレクトリに置く
type Attachment struct {
	ID          int
	TodoID      int
	FileName    string    // アップロードしたときのファイル名
	ContentType string    // 中身から判定した MIME タイプ（例: image/png）
	Size        int64     // バイト数
	StorageName string    `json:"-"` // 保存先のディレクトリでのファイル名（推測できないランダムな名前）。API には出さない
	CreatedAt   time.Time // 添付した日時（UTC）
}

// inlineImageTypes はブラウザでそのまま表示してよい画像の種類。
// SVG はスクリプトを含められるので、表示せずダウンロードさせる
var inlineImageTypes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/gif":  true,
	"image/webp": true,
	"image/bmp":  true,
}

// IsInlineImage はブラウザでサムネイルとして表示できる画像かを返す
func (a Attachment) IsInlineImage() bool {
	return inlineImageTypes[a.ContentType]
}

// NormalizeAttachmentName はファイル名からディレクトリ部分と前後の空白を除き、長すぎる名前は末尾を切る。
// 空になる場合は "file" にする
func NormalizeAttachmentName(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.TrimSpace(filepath.Base(name))
	if name == "" || name == "." || name == "/" {
		return "file"
	}
	if utf8.RuneCountInString(name) > attachmentNameMax {
		name = string([]rune(name)[:attachmentNameMax])
	}
	return name
}
