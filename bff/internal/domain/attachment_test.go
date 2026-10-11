package domain

import (
	"strings"
	"testing"
)

func TestNormalizeAttachmentName(t *testing.T) {
	tests := map[string]string{
		"screenshot.png":          "screenshot.png",
		"  議事録.pdf ":              "議事録.pdf",
		"../../etc/passwd":        "passwd",
		`C:\Users\me\report.xlsx`: "report.xlsx",
		"":                        "file",
		"/":                       "file",
	}
	for in, want := range tests {
		if got := NormalizeAttachmentName(in); got != want {
			t.Errorf("NormalizeAttachmentName(%q) = %q, want %q", in, got, want)
		}
	}
	long := strings.Repeat("あ", 300)
	if got := NormalizeAttachmentName(long); len([]rune(got)) != 255 {
		t.Errorf("long name length = %d", len([]rune(got)))
	}
}

func TestAttachment_IsInlineImage(t *testing.T) {
	for contentType, want := range map[string]bool{
		"image/png": true, "image/jpeg": true, "image/svg+xml": false, "text/html; charset=utf-8": false, "application/pdf": false,
	} {
		if got := (Attachment{ContentType: contentType}).IsInlineImage(); got != want {
			t.Errorf("IsInlineImage(%q) = %v", contentType, got)
		}
	}
}
