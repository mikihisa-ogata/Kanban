package domain

import (
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// 時間の単位を分で表したもの。Jira と同じく 1 日は 8 時間、1 週は 5 日として数える
const (
	MinutesPerHour = 60
	MinutesPerDay  = 8 * MinutesPerHour
	MinutesPerWeek = 5 * MinutesPerDay
)

// durationMax は扱える時間の上限（分）。約 4166 週
const durationMax = 10_000_000

var ErrInvalidDuration = errors.New("時間は「1w 2d 3h 30m」の形式で入力してください（w=週・d=日・h=時間・m=分、1d=8h、1w=5d）")

// durationToken は「数値＋単位」の 1 つ分。数値は小数も使える（例: 1.5h）
var durationToken = regexp.MustCompile(`^(\d+(?:\.\d+)?)([wdhm])`)

var durationUnits = map[string]float64{
	"w": MinutesPerWeek,
	"d": MinutesPerDay,
	"h": MinutesPerHour,
	"m": 1,
}

// ParseDuration は「1w 2d 3h 30m」形式の文字列を分に変える。
// 単位の順番や空白の有無は問わず、大文字も受け付ける（例: "3h30m"、"2D"）。同じ単位が複数あれば足す。
// 小数は分に丸める（例: "1.5h" は 90 分）。単位のない数値や空文字は ErrInvalidDuration
func ParseDuration(s string) (int, error) {
	rest := strings.ToLower(strings.TrimSpace(s))
	if rest == "" {
		return 0, ErrInvalidDuration
	}
	total := 0.0
	for rest != "" {
		match := durationToken.FindStringSubmatch(rest)
		if match == nil {
			return 0, ErrInvalidDuration
		}
		value, err := strconv.ParseFloat(match[1], 64)
		if err != nil {
			return 0, ErrInvalidDuration
		}
		total += value * durationUnits[match[2]]
		rest = strings.TrimLeft(rest[len(match[0]):], " \t")
	}
	minutes := math.Round(total)
	if minutes > durationMax {
		return 0, ErrInvalidDuration
	}
	return int(minutes), nil
}

// FormatDuration は分を「1w 2d 3h 30m」形式にする。0 の単位は省き、0 分は "0m" にする
func FormatDuration(minutes int) string {
	if minutes <= 0 {
		return "0m"
	}
	parts := []string{}
	for _, unit := range []struct {
		size  int
		label string
	}{{MinutesPerWeek, "w"}, {MinutesPerDay, "d"}, {MinutesPerHour, "h"}, {1, "m"}} {
		if n := minutes / unit.size; n > 0 {
			parts = append(parts, strconv.Itoa(n)+unit.label)
			minutes -= n * unit.size
		}
	}
	return strings.Join(parts, " ")
}
