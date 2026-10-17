package domain

import (
	"sort"
	"strings"
)

// BulkIssueChange は一括変更で 1 件の課題に書き込む内容
type BulkIssueChange struct {
	// Todo は変更後の課題。ステータス・完了・優先度・担当者・ラベル・スプリント・エピック・修正バージョンを書き込む
	Todo Todo
	// History はこの課題の変更履歴（変更者は各 entry の UserID）
	History []HistoryEntry
	// Watcher は新しく担当者になり、ウォッチャーに加えるユーザー。0 なら加えない
	Watcher int
}

// PlanningNames は一括変更の履歴にスプリントと修正バージョンの名前を残すための、ID から名前への対応
type PlanningNames struct {
	Sprints  map[int]string
	Versions map[int]string
}

// DiffPlanning は before から after へのスプリントと修正バージョンの変更を、項目ごとの履歴にして返す（ID と日時は付けない）。
// スプリントの 0 はバックログ（なし）。修正バージョンは名前順に並べて比べる
func DiffPlanning(before, after Todo, names PlanningNames) []HistoryEntry {
	entries := []HistoryEntry{}
	add := func(field string, oldValue, newValue *string) {
		if equalValue(oldValue, newValue) {
			return
		}
		entries = append(entries, HistoryEntry{TodoID: after.ID, Field: field, OldValue: oldValue, NewValue: newValue})
	}
	add(HistoryFieldSprint, nameText(names.Sprints, before.SprintID), nameText(names.Sprints, after.SprintID))
	add(HistoryFieldFixVersions, versionsText(names.Versions, before.FixVersionIDs), versionsText(names.Versions, after.FixVersionIDs))
	return entries
}

// versionsText は修正バージョンの名前を名前順に「, 」でつないで返す。なければ nil
func versionsText(names map[int]string, ids []int) *string {
	list := []string{}
	for _, id := range ids {
		list = append(list, *nameText(names, id))
	}
	sort.Strings(list)
	return text(strings.Join(list, ", "))
}
