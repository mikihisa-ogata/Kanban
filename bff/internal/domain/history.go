package domain

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// 履歴の項目名。API の課題の項目（PUT /todos/:id のリクエスト）に合わせる
const (
	HistoryFieldCreated     = "created" // 課題の作成。変更前・変更後は持たない
	HistoryFieldTitle       = "title"
	HistoryFieldDescription = "description"
	HistoryFieldStatus      = "status"
	HistoryFieldDone        = "done"
	HistoryFieldIssueType   = "issueType"
	HistoryFieldPriority    = "priority"
	HistoryFieldLabels      = "labels"
	HistoryFieldStoryPoints = "storyPoints"
	HistoryFieldDeadline    = "deadline"
	HistoryFieldEpic        = "epic"
	HistoryFieldSpace       = "space"
	HistoryFieldKey         = "key"
)

// HistoryEntry は課題の変更履歴の 1 件。値は変更時点の表示用の文字列で、nil は「なし」。
// ステータス（status）は Open などの値のまま、ストーリーポイント（storyPoints）は数値の文字列で持つので、
// 日時と合わせて、ある時点のステータスとポイントを後から再計算できる
type HistoryEntry struct {
	ID        int
	TodoID    int
	Field     string
	OldValue  *string
	NewValue  *string
	CreatedAt time.Time // 記録した日時（UTC）
}

// HistoryNames は履歴にエピックとスペースの名前を残すための、ID から名前への対応
type HistoryNames struct {
	Epics  map[int]string
	Spaces map[int]string
}

// DiffTodo は before から after への変更を、項目ごとの履歴にして返す（ID と日時は付けない）。変更がなければ空
func DiffTodo(before, after Todo, names HistoryNames) []HistoryEntry {
	entries := []HistoryEntry{}
	add := func(field string, oldValue, newValue *string) {
		if equalValue(oldValue, newValue) {
			return
		}
		entries = append(entries, HistoryEntry{TodoID: after.ID, Field: field, OldValue: oldValue, NewValue: newValue})
	}
	add(HistoryFieldTitle, text(before.Title), text(after.Title))
	add(HistoryFieldDescription, text(before.Description), text(after.Description))
	add(HistoryFieldStatus, text(string(before.Status)), text(string(after.Status)))
	add(HistoryFieldDone, boolText(before.Done), boolText(after.Done))
	add(HistoryFieldIssueType, text(string(before.IssueType)), text(string(after.IssueType)))
	add(HistoryFieldPriority, text(string(before.Priority)), text(string(after.Priority)))
	add(HistoryFieldLabels, text(strings.Join(before.Labels, ", ")), text(strings.Join(after.Labels, ", ")))
	add(HistoryFieldStoryPoints, pointsText(before.StoryPoints), pointsText(after.StoryPoints))
	add(HistoryFieldDeadline, text(before.Deadline), text(after.Deadline))
	add(HistoryFieldEpic, nameText(names.Epics, before.EpicID), nameText(names.Epics, after.EpicID))
	add(HistoryFieldSpace, nameText(names.Spaces, before.SpaceID), nameText(names.Spaces, after.SpaceID))
	add(HistoryFieldKey, text(before.Key), text(after.Key))
	return entries
}

// text は空文字を nil（なし）にする
func text(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func boolText(b bool) *string {
	s := strconv.FormatBool(b)
	return &s
}

func pointsText(points *float64) *string {
	if points == nil {
		return nil
	}
	s := strconv.FormatFloat(*points, 'f', -1, 64)
	return &s
}

// nameText はエピック・スペースの名前を返す。0 は nil（なし）、名前が分からなければ「ID 3」のようにする
func nameText(names map[int]string, id int) *string {
	if id == 0 {
		return nil
	}
	if name, ok := names[id]; ok {
		return &name
	}
	s := fmt.Sprintf("ID %d", id)
	return &s
}

func equalValue(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
