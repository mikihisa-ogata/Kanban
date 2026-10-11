package service

import (
	"errors"
	"testing"
	"time"

	"todo-api/internal/domain"
)

func TestDurationUpdate_Parse(t *testing.T) {
	tests := []struct {
		value string
		want  *int
	}{
		{"", nil},
		{"   ", nil},
		{"1w 2d 3h 30m", ptr(2400 + 960 + 180 + 30)},
		{"90m", ptr(90)},
		{"0m", ptr(0)},
	}
	for _, tt := range tests {
		got, err := DurationUpdate{Set: true, Value: tt.value}.parse()
		if err != nil || !equalMinutes(got, tt.want) {
			t.Errorf("parse(%q) = %v, %v, want %v", tt.value, deref(got), err, deref(tt.want))
		}
	}
	if _, err := (DurationUpdate{Set: true, Value: "3"}).parse(); !errors.Is(err, domain.ErrInvalidDuration) {
		t.Errorf("parse(\"3\") error = %v", err)
	}
}

func TestNewWorklog(t *testing.T) {
	start := time.Date(2026, 10, 11, 1, 0, 0, 0, time.UTC)
	w, err := newWorklog("1h 30m", start, "メモ")
	if err != nil || w.Minutes != 90 || !w.StartedAt.Equal(start) || w.Note != "メモ" {
		t.Errorf("newWorklog = %+v, %v", w, err)
	}
	for _, timeSpent := range []string{"0m", "", "1", "1x"} {
		if _, err := newWorklog(timeSpent, start, ""); !errors.Is(err, ErrInvalidWorklogTime) {
			t.Errorf("newWorklog(%q) error = %v", timeSpent, err)
		}
	}
	if _, err := newWorklog("1h", time.Time{}, ""); !errors.Is(err, ErrInvalidWorklogStart) {
		t.Errorf("zero start error = %v", err)
	}
}

func ptr(n int) *int { return &n }

func deref(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}
